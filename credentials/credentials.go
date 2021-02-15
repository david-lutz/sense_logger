package credentials

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/buger/jsonparser"
)

const authenticateURL = "https://api.sense.com/apiservice/api/v1/authenticate"

// Credentials holds the Sense Bearer Token Credentials and MonitorId
type Credentials struct {
	Token     string    `json:"token"`
	MonitorID int64     `json:"monitorId"`
	TimeZone  string    `json:"timeZone"`
	Timestamp time.Time `json:"timestamp"`
}

// WriteCreds saves the Credentials to a file
func WriteCreds(credentials Credentials, filename string) error {
	log.Println("WriteCreds", credentials)

	data, err := json.MarshalIndent(credentials, "", "  ")
	if err != nil {
		return err
	}

	data = append(data, 10)                      // Add newline character to make file prettier
	err = ioutil.WriteFile(filename, data, 0600) // Make file permissions read+write for user only
	log.Println("write err:", err)

	return err
}

// ReadCreds reads Credentials from a file
func ReadCreds(filename string) (Credentials, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return Credentials{}, err
	}

	var creds Credentials
	err = json.Unmarshal(data, &creds)
	return creds, err
}

// FetchCredentials gets bearer token and monitor credentials from Sense Web Service
func FetchCredentials(email, password string) (Credentials, error) {
	res, err := http.PostForm(authenticateURL,
		url.Values{
			"email":    {email},
			"password": {password},
		})
	if err != nil {
		return Credentials{}, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		err = fmt.Errorf("StatusCode: %d, Status: %s", res.StatusCode, res.Status)
		return Credentials{}, err
	}

	data, err := ioutil.ReadAll(res.Body)

	authData := make(map[string]interface{})
	err = json.Unmarshal(data, &authData)
	if err != nil {
		return Credentials{}, err
	}

	// User jsonparser to extract selected fields from JSON response...
	token, err := jsonparser.GetString(data, "access_token")
	if err != nil {
		return Credentials{}, err
	}

	monitorID, err := jsonparser.GetInt(data, "monitors", "[0]", "id")
	if err != nil {
		return Credentials{}, err
	}

	timeZone, err := jsonparser.GetString(data, "monitors", "[0]", "time_zone")
	if err != nil {
		return Credentials{}, err
	}

	creds := Credentials{
		Token:     token,
		MonitorID: monitorID,
		TimeZone:  timeZone,
		Timestamp: time.Now().UTC(),
	}

	return creds, err
}
