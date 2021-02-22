package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/david-lutz/sense_logger/credentials"
	"github.com/david-lutz/sense_logger/sense"
	"github.com/gorilla/websocket"
	"golang.org/x/time/rate"
)

var wsURL = "wss://clientrt.sense.com/monitors/%d/realtimefeed?access_token=%s"

// Launch a webSocket reader for realtime Sense data.  The Sense website has a tendency to
// uncerimoniously disconnect every so often, so we catch the disconnects and reconnect.
func senseReader(creds credentials.Credentials, publishers ...publisher) {
	// Rate limiter so we don't reconnect to fast
	limiter := rate.NewLimiter(rate.Every(30*time.Second), 3)

	for {
		limiter.Reserve()
		limiter.Wait(context.Background())
		webSocketReader(creds, publishers...)
	}
}

// Connect to the WebSocket endpoint and read until loop finishes (i.e. Sense closes the connection)
func webSocketReader(creds credentials.Credentials, publishers ...publisher) {
	url := fmt.Sprintf(wsURL, creds.MonitorID, creds.Token)
	log.Printf("Connecting to %s", url)

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if conn != nil {
		defer conn.Close()
	}
	if err != nil {
		log.Println("WebSocket Dial():", err)
		return
	}

	// Sense WebSocket Read Loop
	done := make(chan error)
	go webSocketReadLoop(conn, done, publishers...)

	// Wait for read loop to finish
	<-done
}

// Read and parse messages from the websocket, dispatching them to publishers
// Will close the done channel when the ReadMessage loop exits
func webSocketReadLoop(wsConn *websocket.Conn, done chan error, publishers ...publisher) {
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
					for _, publisher := range publishers {
						publisher.Publish(realtime)
					}
				}
			}
		}
	}
}
