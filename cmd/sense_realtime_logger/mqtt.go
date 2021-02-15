package main

import (
	"crypto/tls"
	"log"

	"github.com/david-lutz/sense_logger/config"
	"github.com/david-lutz/sense_logger/sense"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Setup MQTT Connect with clean session and aut-reconnect enabled, username and password are optional
func mqttConnect(mqttCfg config.MQTTConfig) (mqtt.Client, error) {
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
		return nil, nil
	}

	return client, nil
}

// Publish RealTime messages on an MQTT topic
func mqttPublisherLoop(client mqtt.Client, topic string, msgCh chan sense.RealTime, errCh chan errorMsg) {
	for realtime := range msgCh {
		json, err := realtime.ToJSON()
		if err != nil {
			logErrorMsg("mqttPublisherLoop", err, errCh)
			continue
		}

		token := client.Publish(topic, 0, false, json)
		if token.Wait() && token.Error() != nil {
			logErrorMsg("mqttPublisherLoop", err, errCh)
			continue
		}

	}
}

// Log Connections
func mqttLogConnection(client mqtt.Client) {
	log.Println("MQTT Connected")
}

// Log Disconnects
func mqttLogDisconnect(client mqtt.Client, err error) {
	log.Println("MQTT Disconnected:", err)
}
