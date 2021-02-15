package main

import (
	"fmt"
	"time"

	"github.com/david-lutz/sense_logger/config"
	"github.com/david-lutz/sense_logger/sense"

	influxdb "github.com/influxdata/influxdb1-client/v2"
)

func influxDBConnect(influxCfg *config.InfluxHDBTTPConfig) (influxdb.Client, error) {
	client, err := influxdb.NewHTTPClient(influxdb.HTTPConfig{
		Addr:     influxCfg.Addr,
		Username: influxCfg.Username,
		Password: influxCfg.Password,
		Timeout:  1 * time.Second, // Data is coming in quickly, so use a small timeout
	})

	return client, err
}

// Store Realtime Messages in InfluxDB
func influxDBPublisherLoop(client influxdb.Client, batchCfg *config.InfluxDBBatchConfig, monitorID int64,
	threshold float64, msgCh chan sense.RealTime, errCh chan errorMsg) {

	for realtime := range msgCh {
		// Use a Rate Limited for logging
		batch, err := newBatch(batchCfg, monitorID, threshold, realtime)
		if err != nil {
			logErrorMsg("influxDBPublisherLoop", err, errCh)
			continue
		}

		err = client.Write(batch)
		if err != nil {
			logErrorMsg("influxDBPublisherLoop", err, errCh)
			continue
		}
	}
}

// Create an InfluxDB Point from the RealTime value
func newPoint(measurement string, monitorID int64, threshold float64, realtime sense.RealTime) (*influxdb.Point, error) {
	// If the production is below the threshold, set it to zero
	productionCooked := realtime.Production
	if productionCooked < threshold {
		productionCooked = 0.0
	}

	// Tag with MonitorID
	tags := map[string]string{
		"monitorID": fmt.Sprintf("%d", monitorID),
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
	point, err := influxdb.NewPoint("sense_realtime", tags, fields)
	return point, err
}

// Make an InfluxDB batch with a single RealTime data point
func newBatch(batchCfg *config.InfluxDBBatchConfig, monitorID int64, threshold float64, realtime sense.RealTime) (influxdb.BatchPoints, error) {
	batch, err := influxdb.NewBatchPoints(influxdb.BatchPointsConfig{
		Database:        batchCfg.Database,
		RetentionPolicy: batchCfg.RetentionPolicy,
		Precision:       batchCfg.Precision,
	})
	if err != nil {
		return nil, err
	}

	point, err := newPoint(batchCfg.Measurement, monitorID, threshold, realtime)
	if err != nil {
		return nil, err
	}
	batch.AddPoint(point)
	return batch, nil
}
