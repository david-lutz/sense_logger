package main

import (
	"fmt"
	"log"

	"github.com/david-lutz/sense_logger/config"
	"github.com/david-lutz/sense_logger/sense"
	"github.com/jessevdk/go-flags"
)

type publisher interface {
	Publish(sense.RealTime) // Publish a realtime message, should not block
	Close()                 // Final shutdown of underlying publisher resources
}

func main() {
	log.SetFlags(0)

	// Command Line Options
	var opts struct {
		ConfigFile string `short:"c" long:"config" description:"Config file path" default:"~/.sense_logger.toml"`
	}
	_, err := flags.Parse(&opts)
	if err != nil {
		log.Fatal(err)
	}

	// Load Config
	cfg, err := config.LoadConfig(opts.ConfigFile, true)
	if err != nil {
		log.Fatal(err)
	}

	// Connect to MQTT Broker
	mqttPublisher, err := mqttConnect(cfg.MQTT)
	if err != nil {
		log.Fatal("mqttConnect():", err)
	}
	defer mqttPublisher.Close()

	// Connect to InfluxDB
	influxDBPublisher := influxDBConnect(cfg)
	defer influxDBPublisher.Close()

	// WebSocket read loop
	senseReader(cfg.Sense.Credentials, influxDBPublisher, mqttPublisher)
}

// Debuging Publisher
type logPublisher struct{}

func (p *logPublisher) Close() {}
func (p *logPublisher) Publish(realtime sense.RealTime) {
	json, _ := realtime.ToJSON()
	fmt.Println(string(json))
}
