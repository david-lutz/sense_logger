package main

import (
	"crypto/tls"
	"log"
	"time"

	"github.com/david-lutz/sense_logger/config"
	"github.com/david-lutz/sense_logger/sense"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"golang.org/x/time/rate"
)

// Publisher implementation
type mqttPublisher struct {
	client  mqtt.Client
	topic   string
	limiter *rate.Limiter
}

// Setup MQTT Connect with clean session and auto-reconnect enabled, username and password are optional
func mqttConnect(mqttCfg config.MQTTConfig) (publisher, error) {
	connOpts := mqtt.NewClientOptions().AddBroker(mqttCfg.Broker).SetCleanSession(true).SetAutoReconnect(true)
	connOpts.SetOnConnectHandler(mqttLogConnection)
	connOpts.SetConnectionLostHandler(mqttLogDisconnect)
	if mqttCfg.Password != "" {
		connOpts.SetUsername(mqttCfg.Password)
		if mqttCfg.Password != "" {
			connOpts.SetPassword(mqttCfg.Password)
		}
	}
	tlsConfig := &tls.Config{InsecureSkipVerify: true, ClientAuth: tls.NoClientCert}
	connOpts.SetTLSConfig(tlsConfig)

	client := mqtt.NewClient(connOpts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}

	return &mqttPublisher{
		client:  client,
		topic:   mqttCfg.Topic,
		limiter: rate.NewLimiter(rate.Every(30*time.Second), 10),
	}, nil
}

// Log Connections
func mqttLogConnection(client mqtt.Client) {
	log.Println("MQTT Connected")
}

// Log Disconnects
func mqttLogDisconnect(client mqtt.Client, err error) {
	log.Println("MQTT Disconnected:", err)
}

// Close publisher
func (p *mqttPublisher) Close() {
	p.client.Disconnect(1000)
}

// Publish a Realtime data point to MQTT
func (p *mqttPublisher) Publish(realtime sense.RealTime) {
	json, err := realtime.ToJSON()
	if err != nil {
		if p.limiter.Allow() {
			log.Print("MQTT JSON Marshall:", err)
		}
	} else {
		token := p.client.Publish(p.topic, 0, false, json)

		// Async error logging for MQTT Publish
		go func() {
			if token.Wait() && token.Error() != nil {
				if p.limiter.Allow() {
					log.Print("MQTT Publish:", err)
				}
			}
		}()
	}
}
