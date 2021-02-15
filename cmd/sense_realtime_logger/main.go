package main

import (
	"fmt"
	"log"

	"github.com/david-lutz/sense_logger/config"
	"github.com/david-lutz/sense_logger/sense"
	"github.com/gorilla/websocket"
	"github.com/jessevdk/go-flags"
)

var outFileName = "realtime.json"
var monitorID = "44610"
var accessToken = "t1.30148.29985.47c565942d8c21cab80bc30de33e732a1e8b3f95225b2973e6f49c2ab0c70e03"

var wsURL = "wss://clientrt.sense.com/monitors/%d/realtimefeed?access_token=%s"

// Read and parse messages from the websocket, dispatching them to MQTT and InfluxDB publishers
func webSocketReadLoop(wsConn *websocket.Conn, mqttCh, influxCh chan sense.RealTime, errCh chan errorMsg, done chan error) {
	defer close(done)

	for {
		messageType, message, err := wsConn.ReadMessage()
		if err != nil {
			log.Println("WebSocket ReadMessage():", err)
			return
		}
		if messageType == 1 {
			// Process "realtime_update" messages only
			if senseMsgType, err := sense.MessageType(message); err == nil && senseMsgType == "realtime_update" {
				if realtime, err := sense.ParseRealTimeData(message); err == nil {

					// MQTT Publisher
					publishMessage(mqttCh, realtime)

					// InfluxDB Logger
					publishMessage(influxCh, realtime)
				}
			}
		}
	}
}

// Non-blocking publish a RealTime struct on a channel.  If channel is nil no message wll be published.
func publishMessage(ch chan sense.RealTime, msg sense.RealTime) bool {
	if ch == nil {
		return false
	}

	select {
	case ch <- msg:
		return true
	default:
		return false
	}
}

func senseReader(cfg *config.Config, mqttCh, influxCh chan sense.RealTime, errCh chan errorMsg) {

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

	// Error channel, has some buffer included
	errCh := make(chan errorMsg, 50)

	// MQTT Connection
	mqttClient, err := mqttConnect(cfg.MQTT)
	if err != nil {
		log.Fatal("mqttConnect():", err)
	}

	// MQTT Publisher Loop
	mqttCh := make(chan sense.RealTime)
	go mqttPublisherLoop(mqttClient, "sense/realtime", mqttCh, errCh)

	// InfluxDB Connection
	influxClient, err := influxDBConnect(&cfg.InfluxDB.HTTPConfig)
	if err != nil {
		log.Fatal("influxDBConnect():", err)
	}

	// InfluxDB Publisher Loop
	influxCh := make(chan sense.RealTime)
	go influxDBPublisherLoop(influxClient, &cfg.InfluxDB.RealTime, cfg.Sense.Credentials.MonitorID, cfg.Sense.ProductionThreshold, influxCh, errCh)

	// Sense Connection
	senseReader(cfg, mqttCh, influxCh, errCh)
	url := fmt.Sprintf(wsURL, cfg.Sense.Credentials.MonitorID, cfg.Sense.Credentials.Token)
	log.Printf("Connecting to %s", url)

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	defer conn.Close()
	if err != nil {
		logErrorMsg("senseReader", err, errCh)
		return
	}

	// Sense WebSocket Read Loop
	done := make(chan error)
	go webSocketReadLoop(conn, mqttCh, influxCh, errCh, done)

	// Wait for read loop to finish
	<-done

	// Try to gracefully close the MQTT connection
	mqttClient.Disconnect(1000) // 1 Second Quiece
}
