package parser

import (
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/sqooba/go-common/logging"
)

var (
	log = logging.NewLogger()

	regexTakeoff  = regexp.MustCompile(`⛳ (.*?) \[`)
	regexCountry  = regexp.MustCompile(`\[([A-Z]{2})\]`)
	regexDuration = regexp.MustCompile(`⌛ ([0-9:].*?) ∷`)
	regexSpeed    = regexp.MustCompile(`ø (.*?) km/h ∷`)
	regexAltitude = regexp.MustCompile(`⊺ (.*?) m`)
)

// Flight represents a flight.
type Flight struct {
	FullName        string  `json:"full_name"`
	FlightDate      int64   `json:"flight_date"`
	Distance        float64 `json:"distance"`
	FlightType      string  `json:"flight_type"`
	PublicationDate int64   `json:"publication_date"`
	Url             string  `json:"url"`
	TakeOff         string  `json:"take_off"`
	CountryCode     string  `json:"country_code"`
	AverageSpeed    float64 `json:"average_speed"`
	FlightDuration  string  `json:"flight_duration"`
	AltitudeMax     int64   `json:"altitude_max"`
	ParsingSource   string  `json:"parsing_source"`
}

func GetFlightInfo(url string, source string) (*Flight, error) {
	flight := Flight{ParsingSource: source}
	response, err := http.Get(url)
	if err != nil {
		log.Errorf("Error reading url: %v", err)
		return nil, err
	}
	log.Tracef("HTTP response: %s", response.Body)
	defer response.Body.Close()

	doc, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		log.Errorf("Error loading HTTP response body: %v", err)
		return nil, err
	}

	matches := doc.Find("meta[property*='og:description']")
	if matches.Length() > 0 {
		row, _ := matches.First().Attr("content")
		log.Tracef("Extracted row: %v", row)
		flight.TakeOff = regexTakeoff.FindStringSubmatch(row)[1]
		flight.CountryCode = regexCountry.FindStringSubmatch(row)[1]
		flight.FlightDuration = regexDuration.FindStringSubmatch(row)[1]
		speed, err := strconv.ParseFloat(regexSpeed.FindStringSubmatch(row)[1], 64)
		if err != nil {
			log.Errorf("Error parsing speed: %v", err)
			return nil, err
		}
		flight.AverageSpeed = speed
		altitude, err := strconv.ParseInt(regexAltitude.FindStringSubmatch(row)[1], 10, 64)
		if err != nil {
			log.Errorf("Error parsing altitude: %v", err)
			return nil, err
		}
		flight.AltitudeMax = altitude
		return &flight, nil
	}
	return nil, err
}

// ParseDate parse a date using multiple formats.
func ParseDate(input string) (time.Time, error) {
	flightDateLayouts := [2]string{"02.01.2006", "2.01.2006"}
	for _, format := range flightDateLayouts {
		t, err := time.Parse(format, input)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, errors.New("unrecognized time format")
}
