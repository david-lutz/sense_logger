package main

import (
	"fmt"
	"log"
	"time"

	"github.com/david-lutz/sense_logger/config"
	influxdb "github.com/influxdata/influxdb1-client/v2"
	"github.com/jessevdk/go-flags"
)

// Simple struct to hold pertinent details from the PGE Bill measurement
type bill struct {
	// Leave as an interface{}, we are just going to use it in Sprintf so we don't want to do too much parsing
	time interface{}
	bank interface{}
}

func main() {
	// Command Line Options
	var opts struct {
		ConfigFile string `short:"c" long:"config" description:"Config file path" default:"~/.sense_logger.toml"`
		BackFill   bool   `short:"b" long:"backfill" description:"Default only calculates from the latest bill date, use this flag to backfill all data."`
	}
	_, err := flags.Parse(&opts)
	fatalOnErr(err)

	// Load Config
	cfg, err := config.LoadConfig(opts.ConfigFile, true)
	fatalOnErr(err)

	// Create Influx Client
	client, err := influxdb.NewHTTPClient(influxdb.HTTPConfig{
		Addr:     cfg.InfluxDB.HTTPConfig.Addr,
		Username: cfg.InfluxDB.HTTPConfig.Username,
		Password: cfg.InfluxDB.HTTPConfig.Password,
		Timeout:  30 * time.Second,
	})
	fatalOnErr(err)
	defer client.Close()

	// Query the PGE Bill measurement
	bills, err := queryBills(cfg, client, opts.BackFill)
	fatalOnErr(err)

	// Generate and run the update queries
	queries, err := generateQueries(cfg, bills)
	fatalOnErr(err)
	for _, query := range queries {
		response, err := client.Query(query)
		fatalOnErr(err)
		fatalOnErr(response.Error())
		// Don't bother parsing results, we already Fataled if there was a problem
	}
}

// Query the PGE Bill table (if backfilling we get the entire table, no time cutoff yet)
func queryBills(cfg *config.Config, client influxdb.Client, backfill bool) ([]bill, error) {
	// Make the query, if not backfilling we only get the last bill record
	var queryStr string
	if backfill {
		queryStr = fmt.Sprintf("SELECT bank FROM %s",
			measurement(cfg.InfluxDB.PGEBill.RetentionPolicy, cfg.InfluxDB.PGEBill.Measurement))
	} else {
		queryStr = fmt.Sprintf("SELECT last(bank) as bank FROM %s",
			measurement(cfg.InfluxDB.PGEBill.RetentionPolicy, cfg.InfluxDB.PGEBill.Measurement))
	}
	query := influxdb.NewQuery(queryStr, cfg.InfluxDB.PGEBill.Database, "ns")

	// Query and parse results
	if response, err := client.Query(query); err == nil && response.Error() == nil {
		// Make sure we only have one result with one series)
		if len(response.Results) > 1 || len(response.Results[0].Series) > 1 {
			return nil, fmt.Errorf("PGE BIll query returned more than one result/series")
		}

		timeIdx, bankIdx := columnIndexes(response.Results[0].Series[0].Columns)
		bills := make([]bill, len(response.Results[0].Series[0].Values))
		for i, value := range response.Results[0].Series[0].Values {
			bills[i] = bill{time: value[timeIdx], bank: value[bankIdx]}
		}
		return bills, nil
	} else if err != nil {
		return nil, err
	} else {
		return nil, response.Error()
	}
}

// Generate SELECT ... INTO queries based on the bill info
func generateQueries(cfg *config.Config, bills []bill) ([]influxdb.Query, error) {
	queries := make([]influxdb.Query, len(bills))
	for i, b := range bills {
		var timeClause string
		if i+1 < len(bills) {
			// Not the last bill
			timeClause = fmt.Sprintf("time >= %v and time < %v", b.time, bills[i+1].time)
		} else {
			// The last bill
			timeClause = fmt.Sprintf("time >= %v", b.time)
		}

		q := fmt.Sprintf("select %v + cumulative_sum(\"net\") as bank_estimate into "+
			"%s from (select \"production\"-\"consumption\" as \"net\" from %s where %s) group by *;",
			b.bank,
			measurement(cfg.InfluxDB.PGEBill.RetentionPolicy, cfg.InfluxDB.PGEBank.Measurement),
			measurement(cfg.InfluxDB.Month.RetentionPolicy, cfg.InfluxDB.Month.Measurement),
			timeClause)

		queries[i] = influxdb.NewQuery(q, cfg.InfluxDB.PGEBank.Database, "ns")
	}
	return queries, nil
}

func measurement(retentionPolicy, measurement string) string {
	if retentionPolicy != "" {
		return fmt.Sprintf("\"%s\".\"%s\"", retentionPolicy, measurement)
	} else {
		return fmt.Sprintf("\"%s\"", measurement)
	}
}

func columnIndexes(columns []string) (timeIdx, bankIdx int) {
	for i, col := range columns {
		switch col {
		case "time":
			timeIdx = i
		case "bank":
			bankIdx = i
		}
	}
	return
}

func fatalOnErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
