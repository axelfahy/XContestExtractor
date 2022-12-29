package parser

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
)

func TestGetFlightInfo01(t *testing.T) {
	url := "https://www.xcontest.org/world/en/flights/detail:Claricegomes/5.12.2021/14:23"
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	content, err := os.ReadFile(filepath.Join("testdata", "flight_detail_01.html"))
	if err != nil {
		t.Errorf("Error reading file: %v", err)
	}
	httpmock.RegisterResponder("GET", url,
		httpmock.NewStringResponder(200, string(content)))
	flight, err := GetFlightInfo(url, "test")
	if err != nil {
		t.Errorf("Error getting flight information: %v", err)
	}
	if flight.CountryCode != "BR" {
		t.Errorf("Retrieved country code is wrong: %s", flight.CountryCode)
	}
}

func TestGetFlightInfo02(t *testing.T) {
	url := "https://www.xcontest.org/world/en/flights/detail:Fayber/5.12.2021/17:01"
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	content, err := os.ReadFile(filepath.Join("testdata", "flight_detail_02.html"))
	if err != nil {
		t.Errorf("Error reading file: %v", err)
	}
	httpmock.RegisterResponder("GET", url,
		httpmock.NewStringResponder(200, string(content)))
	flight, err := GetFlightInfo(url, "test")
	if err != nil {
		t.Errorf("Error getting flight information: %v", err)
	}
	if flight.CountryCode != "CO" {
		t.Errorf("Retrieved country code is wrong: %s", flight.CountryCode)
	}
}

func TestGetFlightInfo03(t *testing.T) {
	url := "https://www.xcontest.org/world/en/flights/detail:HENRYHOYOS/5.12.2021/19:11"
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	content, err := os.ReadFile(filepath.Join("testdata", "flight_detail_03.html"))
	if err != nil {
		t.Errorf("Error reading file: %v", err)
	}
	httpmock.RegisterResponder("GET", url,
		httpmock.NewStringResponder(200, string(content)))
	flight, err := GetFlightInfo(url, "test")
	if err != nil {
		t.Errorf("Error getting flight information: %v", err)
	}
	if flight.CountryCode != "CO" {
		t.Errorf("Retrieved country code is wrong: %s", flight.CountryCode)
	}
	if flight.TakeOff != "?" {
		t.Errorf("Retrieved take-off is wrong: %s", flight.TakeOff)
	}
}

func TestParseDate01(t *testing.T) {
	input := "21.09.2021"
	result := time.Date(2021, 9, 21, 0, 0, 0, 0, time.UTC)

	date, err := ParseDate(input)
	if err != nil {
		t.Errorf("Error parsing the date: %v", err)
	}
	if result != date {
		t.Errorf("Parsing date is wrong: %s != %s", result, date)
	}
}

func TestParseDate02(t *testing.T) {
	input := "1.09.2021"
	result := time.Date(2021, 9, 1, 0, 0, 0, 0, time.UTC)

	date, err := ParseDate(input)
	if err != nil {
		t.Errorf("Error parsing the date: %v", err)
	}
	if result != date {
		t.Errorf("Parsing date is wrong: %s != %s", result, date)
	}
}
