package config

/*
 * This file defines the configuration structures and has a function for loading config.
 */

import (
	"io/ioutil"

	"github.com/david-lutz/sense_logger/credentials"
	"github.com/mitchellh/go-homedir"
	"github.com/pelletier/go-toml"
)

// SenseConfig holds Sense Monitor parameters
type SenseConfig struct {
	CredentialFile      string  `toml:"credential-file"`
	ProductionThreshold float64 `toml:"production_threshold"`
	Credentials         credentials.Credentials
}

// MQTTConfig holds broker and topic options for MQTT publishing
type MQTTConfig struct {
	Broker   string `toml:"broker"`
	Username string `toml:"username"`
	Password string `toml:"password"`
	Topic    string `toml:"topic"`
}

// InfluxHDBTTPConfig holds database connection parameters
type InfluxHDBTTPConfig struct {
	Addr     string `toml:"addr"`
	Username string `toml:"username"`
	Password string `toml:"password"`
}

// InfluxDBBatchConfig holds configuration for a batch of points
type InfluxDBBatchConfig struct {
	Database        string `toml:"database"`
	RetentionPolicy string `toml:"retention_policy"`
	Measurement     string `toml:"measurement"`
	Precision       string `toml:"precision"`
}

// InfluxDBConfig holds database and measurement parameters
type InfluxDBConfig struct {
	HTTPConfig InfluxHDBTTPConfig  `toml:"Connection"`
	Hour       InfluxDBBatchConfig `toml:"Hour"`
	Day        InfluxDBBatchConfig `toml:"Day"`
	Month      InfluxDBBatchConfig `toml:"Month"`
	Year       InfluxDBBatchConfig `toml:"Year"`
	RealTime   InfluxDBBatchConfig `toml:"RealTime"`
	PGEBill    InfluxDBBatchConfig `toml:"PGEBill"`
	PGEBank    InfluxDBBatchConfig `toml:"PGEBank"`
}

// Config is the structure of the external configuration file
type Config struct {
	Sense    SenseConfig    `toml:"Sense"`
	MQTT     MQTTConfig     `toml:"MQTT"`
	InfluxDB InfluxDBConfig `toml:"InfluxDB"`
}

// LoadConfig loads config from file and optionally loads Sense credentials
func LoadConfig(configFile string, loadCredentials bool) (*Config, error) {
	filename, err := homedir.Expand(configFile)
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Config
	err = toml.Unmarshal(data, &config)

	// Go ahead and load Sense Credentials
	if loadCredentials {
		credFile, err := homedir.Expand(config.Sense.CredentialFile)
		if err != nil {
			return nil, err
		}
		creds, err := credentials.ReadCreds(credFile)
		if err != nil {
			return nil, err
		}
		config.Sense.Credentials = creds
	}

	return &config, err
}
