package main

import (
	"fmt"
	"log"
	"time"

	"github.com/david-lutz/sense_logger/config"
	"github.com/david-lutz/sense_logger/sense"
	"golang.org/x/time/rate"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
)

// Publisher implementation
type influxDBPublisher struct {
	client      influxdb2.Client
	writeAPI    api.WriteAPI
	measurement string
	monitorID   int64
	threshold   float64
}

// Setup connection to InfluxDB database for writing realtime data points
func influxDBConnect(cfg *config.Config) publisher {
	client := influxdb2.NewClientWithOptions(
		cfg.InfluxDB.Server.URL,
		cfg.InfluxDB.Server.Token,
		influxdb2.DefaultOptions().SetPrecision(time.Microsecond)) // Precision in Sense message

	writeAPI := client.WriteAPI(cfg.InfluxDB.Server.Org, cfg.InfluxDB.RealTime.Bucket)

	// Limit how fast we can spam the log
	limiter := rate.NewLimiter(rate.Every(30*time.Second), 10)
	go influxDBErrorLogger(writeAPI.Errors(), limiter)

	return &influxDBPublisher{
		client:      client,
		writeAPI:    writeAPI,
		measurement: cfg.InfluxDB.RealTime.Measurement,
		monitorID:   cfg.Sense.Credentials.MonitorID,
		threshold:   cfg.Sense.ProductionThreshold}
}

// Error logging loop for async InfluxDB writes
func influxDBErrorLogger(errCh <-chan error, limiter *rate.Limiter) {
	for err := range errCh {
		if limiter.Allow() {
			log.Print("Influx WriteAPI:", err)
		}
	}
}

// Close publisher
func (p *influxDBPublisher) Close() {
	p.writeAPI.Flush()
	p.client.Close()
}

// Publish a Realtime data point to InfluxDB
func (p *influxDBPublisher) Publish(realtime sense.RealTime) {
	// If the production is below the threshold, set it to zero
	productionCooked := realtime.Production
	if productionCooked < p.threshold {
		productionCooked = 0.0
	}

	// Tag with MonitorID
	tags := map[string]string{
		"monitorID": fmt.Sprintf("%d", p.monitorID),
	}

	// Map structure to InfluxDB fields
	fields := map[string]interface{}{
		"voltageA":            realtime.Voltage[0],
		"voltageB":            realtime.Voltage[1],
		"frequency":           realtime.Frequency,
		"consumptionChannelA": realtime.Channels[0],
		"consumptionChannelB": realtime.Channels[1],
		"productionChannelA":  realtime.Channels[2],
		"productionChannelB":  realtime.Channels[3],
		"consumption":         realtime.Consumption,
		"production":          productionCooked,
		"production_raw":      realtime.Production,
	}

	point := write.NewPoint(p.measurement, tags, fields, realtime.Timestamp)
	p.writeAPI.WritePoint(point)
}
