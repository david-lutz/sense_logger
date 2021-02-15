package sense

import (
	"encoding/json"

	"github.com/buger/jsonparser"
)

// RealTime contains selected fields from the Sense "realtime_uptime" message
type RealTime struct {
	Voltage     [2]float64 `json:"voltage"`
	Frequency   float64    `json:"frequency"`
	Channels    [4]float64 `json:"channels"`
	Consumption float64    `json:"consumption"`
	Production  float64    `json:"production"`
}

// ToJSON converts the RealTime struct to a JSON Object
func (rt RealTime) ToJSON() ([]byte, error) {
	json, err := json.Marshal(rt)
	return json, err
}

// MessageType parses the "type" field from sense websocket message
func MessageType(message []byte) (string, error) {
	messageType, err := jsonparser.GetUnsafeString(message, "type")
	if err != nil {
		return "", err
	}
	return messageType, nil
}

// ParseRealTimeData extracts selected fields from the "realtime_update" message type
func ParseRealTimeData(message []byte) (RealTime, error) {
	results := RealTime{}

	var loopErr error
	jsonparser.EachKey(message,
		func(idx int, value []byte, vt jsonparser.ValueType, err error) {
			if err != nil {
				loopErr = err
				return
			}
			val, err2 := jsonparser.ParseFloat(value)
			if err2 != nil {
				loopErr = err
				return
			}
			switch idx {
			case 0:
				results.Voltage[0] = val
			case 1:
				results.Voltage[1] = val
			case 2:
				results.Channels[0] = val
			case 3:
				results.Channels[1] = val
			case 4:
				results.Channels[2] = -val
			case 5:
				results.Channels[3] = -val
			case 6:
				results.Frequency = val
			case 7:
				results.Consumption = val
			case 8:
				results.Production = val
			}
		},
		[]string{"payload", "voltage", "[0]"},
		[]string{"payload", "voltage", "[1]"},
		[]string{"payload", "channels", "[0]"},
		[]string{"payload", "channels", "[1]"},
		[]string{"payload", "channels", "[2]"},
		[]string{"payload", "channels", "[3]"},
		[]string{"payload", "hz"},
		[]string{"payload", "w"},
		[]string{"payload", "solar_w"})
	if loopErr != nil {
		return RealTime{}, loopErr
	}

	return results, nil
}
