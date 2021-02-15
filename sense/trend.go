package sense

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/buger/jsonparser"
	"github.com/david-lutz/sense_logger/credentials"
)

const trendURL = "https://api.sense.com/apiservice/api/v1/app/history/trends?monitor_id=%d&scale=%s&start=%s&read_combined=true"

// TrendRecord holds one data point from the Sense trend report
type TrendRecord struct {
	Consumption float64
	Production  float64
	Timestamp   time.Time
	Scale       Scale
}

// Parse the "start", "end", and "steps" fields in the response
func parseTimeAndSteps(body []byte) (time.Time, time.Time, int, error) {
	steps, err := jsonparser.GetInt(body, "steps")
	if err != nil {
		return time.Time{}, time.Time{}, 0, err
	}

	startStr, err := jsonparser.GetString(body, "start")
	if err != nil {
		return time.Time{}, time.Time{}, 0, err
	}
	startTime, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		return time.Time{}, time.Time{}, 0, err
	}

	endStr, err := jsonparser.GetString(body, "end")
	if err != nil {
		return time.Time{}, time.Time{}, 0, err
	}
	endTime, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		return time.Time{}, time.Time{}, 0, err
	}

	return startTime, endTime, int(steps), nil
}

// GetTrendData returns the Sense trend data (in what I believe are kWh) for the given start time and Scale.
func GetTrendData(creds credentials.Credentials, scale Scale, start time.Time) ([]TrendRecord, error) {

	// Get the location of the Sense Monitor from the credentials for calculating timestamps
	location, err := time.LoadLocation(creds.TimeZone)
	if err != nil {
		return nil, err
	}

	// Validate scale parameter
	switch scale {
	case Hour, Day, Week, Month, Year:
	default:
		return nil, fmt.Errorf("Invalid Scale: %s", scale)
	}

	// HTTP Request with "Authorization" header set to the credential token
	url := fmt.Sprintf(trendURL, creds.MonitorID, scale, start.Format(time.RFC3339))
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("bearer %s", creds.Token))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("status code error: %d %s", res.StatusCode, res.Status)
	}

	// Slurp in the entire response body
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	// How many "steps" are we expecting in the data
	startTime, endTime, steps, err := parseTimeAndSteps(body)
	if err != nil {
		return nil, err
	}

	// Get the timestamps for each trend record.  Hour and Day scales have regular step
	// durations, but  Week, Month and Year scales can vary
	results := make([]TrendRecord, steps)
	switch scale {
	case Hour, Day:
		stepSize := endTime.Sub(startTime) / time.Duration(steps)
		for i := range results {
			results[i].Timestamp = startTime.Add(time.Duration(i) * stepSize)
			results[i].Scale = scale
		}

	case Week, Month:
		dayStart := beginningOfDay(startTime, location)
		for i := range results {
			results[i].Timestamp = dayStart.AddDate(0, 0, i)
			results[i].Scale = scale
		}

	case Year:
		dayStart := beginningOfDay(startTime, location)
		for i := range results {
			results[i].Timestamp = dayStart.AddDate(0, i, 0)
			results[i].Scale = scale
		}
	}

	// Parse Consumption Values
	index := 0
	var parseErr error
	jsonparser.ArrayEach(body,
		func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
			consumtpion, err := strconv.ParseFloat(string(value), 64)
			if err != nil && parseErr == nil {
				parseErr = err
			}
			results[index].Consumption = consumtpion
			index++
		}, "consumption", "totals")
	if parseErr != nil {
		return nil, parseErr
	}

	// Parse Productions Values
	index = 0
	jsonparser.ArrayEach(body,
		func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
			production, err := strconv.ParseFloat(string(value), 64)
			if err != nil && parseErr == nil {
				parseErr = err
			}
			results[index].Production = production
			index++
		}, "production", "totals")
	if parseErr != nil {
		return nil, parseErr
	}

	return results, nil
}

func beginningOfDay(t time.Time, location *time.Location) time.Time {
	return time.Date(t.Year(),
		t.Month(),
		t.Day(),
		0, 0, 0, 0,
		location)
}
