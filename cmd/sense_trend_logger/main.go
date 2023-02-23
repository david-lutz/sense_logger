package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/david-lutz/sense_logger/config"
	"github.com/david-lutz/sense_logger/sense"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"github.com/jessevdk/go-flags"
)

func main() {
	// Command Line Options
	var opts struct {
		ConfigFile string `short:"c" long:"config" description:"Config file path" default:"~/.sense_logger.toml"`
		Scale      string `short:"s" long:"scale" description:"Scale" choice:"HOUR" choice:"DAY" choice:"MONTH" choice:"YEAR" required:"true"`
		Offset     string `short:"o" long:"offset" description:"Offset from now() for start time"`
		Start      string `short:"t" long:"timestamp" description:"Timestamp in RFC3339 format (defaults to now())"`
		Verbose    bool   `short:"v" long:"verbose" description:"Verbose mode"`
	}
	_, err := flags.Parse(&opts)
	fatalOnErr(err)

	// Scale: Hour, Day, Month, or Year
	scale, err := sense.ParseScale(opts.Scale)
	fatalOnErr(err)

	// Offset, defaults to 0s
	offset := 0 * time.Second
	if opts.Offset != "" {
		offset, err = time.ParseDuration(opts.Offset)
		fatalOnErr(err)
	}

	// Start Time, defaults to now() - offset
	starttime := time.Now().UTC().Add(-1 * offset)
	if opts.Start != "" {
		starttime, err = time.Parse(time.RFC3339, opts.Start)
		starttime = starttime.UTC()
		fatalOnErr(err)
	}

	// Load Config
	cfg, err := config.LoadConfig(opts.ConfigFile, true)
	fatalOnErr(err)

	// Get Trend Data from Sense
	trendRecords, err := sense.GetTrendData(cfg.Sense.Credentials, scale, starttime, opts.Verbose)
	fatalOnErr(err)

	// Get the right config for the scale, if the data points are going to be
	// larger than 1 hour, we don't worry about the the productionThreshold
	var batchCfg config.InfluxDBBatchConfig
	productionThreshold := float64(0)
	switch scale {
	case sense.Hour:
		batchCfg = cfg.InfluxDB.Hour
		productionThreshold = cfg.Sense.ProductionThreshold / 1000.0 / 60.0
	case sense.Day:
		batchCfg = cfg.InfluxDB.Day
		productionThreshold = cfg.Sense.ProductionThreshold / 1000.0
	case sense.Month:
		batchCfg = cfg.InfluxDB.Month
	case sense.Year:
		batchCfg = cfg.InfluxDB.Year
	}

	// Filter out TrendRecords with no data, the Sense API will fill return empty
	// future records when we are part way through a time period
	batch := filterPoints(batchCfg.Measurement, cfg.Sense.Credentials.MonitorID,
		productionThreshold, trendRecords)

	// Write to InfluxDB if we have any data
	if len(batch) > 0 {
		client := influxdb2.NewClientWithOptions(
			cfg.InfluxDB.Server.URL,
			cfg.InfluxDB.Server.Token,
			influxdb2.DefaultOptions().SetPrecision(time.Second))
		defer client.Close()

		writeAPI := client.WriteAPIBlocking(
			cfg.InfluxDB.Server.Org,
			batchCfg.Bucket)

		err := writeAPI.WritePoint(context.Background(), batch...)
		fatalOnErr(err)
	}
}

// Add TrendRecords to a batch if they are non-zero, adjusting the produciton value along the way
func filterPoints(measurement string, monitorID int64, threshold float64, trendRecords []sense.TrendRecord) []*write.Point {

	batch := make([]*write.Point, 0, len(trendRecords))
	for _, trendRecord := range trendRecords {
		// Filter out missing data
		if trendRecord.Consumption == 0 && trendRecord.Production == 0 {
			continue
		}

		// Sense always sees a small amount of Solar Production, even in the middle of the night.
		// The "cooked" production tries to reset these values back to zero.
		cooked := trendRecord.Production
		if cooked < threshold {
			cooked = 0.0
		}

		fields := map[string]interface{}{
			"consumption":    trendRecord.Consumption,
			"raw_production": trendRecord.Production,
			"production":     cooked,
		}

		tags := map[string]string{
			"monitorID": fmt.Sprintf("%d", monitorID),
		}

		point := write.NewPoint(measurement, tags, fields, trendRecord.Timestamp)
		batch = append(batch, point)
	}

	return batch
}

func fatalOnErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
