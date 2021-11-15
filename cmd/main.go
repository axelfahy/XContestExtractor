package main

import (
	"encoding/xml"
	"flag"
	"io"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
)

const (
	// Index to store the entries.
	indexName string = "flight-000001"
	// Url of the RSS feed of XContest.
	url string = "https://www.xcontest.org/rss/flights/?world"
	// Date formats.
	pubDateLayout    string = "Mon, 2 Jan 2006 15:04:05 +0000"
	flightDateLayout string = "02.01.06"
)

var (
	log = logrus.StandardLogger()
)

type envConfig struct {
	// ElasticSearch
	ElasticEndpoint string `envconfig:"ELASTICSEARCH_URL" default:"http://127.0.0.1:9200"`
	ElasticUser     string `envconfig:"ELASTICSEARCH_USERNAME" default:"CHANGEME"`
	ElasticPassword string `envconfig:"ELASTICSEARCH_PASSWORD" default:"CHANGEME"`
	IntervalMin     string `envconfig:"RUN_INTERVAL" default:"60"`
}

// XContestEntry represents the RSS feed.
type XContestEntry struct {
	Channel struct {
		Items []struct {
			Title       string `xml:"title"`
			Link        string `xml:"link"`
			Description string `xml:"description"`
			PubDate     string `xml:"pubDate"`
		} `xml:"item"`
	} `xml:"channel"`
}

// Flight represents a flight.
type Flight struct {
	FullName        string `json:"full_name"`
	FlightDate      int64  `json:"flight_date"`
	Distance        string `json:"distance"`
	FlightType      string `json:"flight_type"`
	Url             string `json:"url"`
	PublicationDate int64  `json:"publication_date"`
}

// ExtractFlights extracts the flights from the XML.
func ExtractFlights(body []byte) (*XContestEntry, error) {
	data := &XContestEntry{}
	err := xml.Unmarshal(body, data)
	if err != nil {
		log.Errorf("Error unmarshaling the xml data: %v", err)
		return nil, err
	}
	return data, nil
}

func main() {
	log.Info("Starting XContestRSSExtractor...")

	flag.Parse()

	// Regex to parse flight info.
	regexDistance := regexp.MustCompile(`\[(\d+\.\d+) km`)
	regexFlightType := regexp.MustCompile(`:: (\w+)]`)
	regexFullName := regexp.MustCompile(`\] (.*)`)

	// Loading env variables.
	var env envConfig
	if err := envconfig.Process("", &env); err != nil {
		log.Fatalf("Failed to process env var: %v", err)
	}
	log.Infof("Elastic endpoint     : %s", env.ElasticEndpoint)
	log.Infof("Elastic user         : %s", env.ElasticUser)
	log.Infof("Elastic password     : %s", env.ElasticPassword)
	log.Infof("Running interval [m] : %s", env.IntervalMin)

	interval, err := strconv.Atoi(env.IntervalMin)
	if err != nil {
		log.Fatalf("Error converting the given interval: %v", err)
	}
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	ticker := time.NewTicker(time.Duration(interval) * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case t := <-ticker.C:
			log.Infof("Running extractor at %v", t)

			// Initialization of the ElasticSearch client.
			manager := NewElasticManager(
				env.ElasticEndpoint,
				env.ElasticUser,
				env.ElasticPassword,
				indexName,
			)

			// Read the RSS feed.
			resp, err := http.Get(url)
			if err != nil {
				log.Fatalf("Error requesting url: %v", err)
			}
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Fatalf("Error reading the body: %v", err)
			}

			// Extract the flights.
			flights, err := ExtractFlights(body)
			if err != nil {
				log.Fatalf("Error extracting flights from XML: %v", err)
			}
			numInsertion := 0
			// Insert each flight into ES.
			for i, entry := range flights.Channel.Items {
				log.Debugf("Processing flight: %s (%d / %d)", entry, i, len(flights.Channel.Items))
				publicationDate, err := time.Parse(pubDateLayout, entry.PubDate)
				if err != nil {
					log.Fatalf("Error converting publication date to timestamp: %v", err)
				}
				date, err := time.Parse(flightDateLayout, strings.Split(entry.Title, " ")[0])
				if err != nil {
					log.Fatalf("Error converting date flight to timestamp: %v", err)
				}
				flight := Flight{
					regexFullName.FindStringSubmatch(entry.Title)[1],
					date.UnixMilli(),
					regexDistance.FindStringSubmatch(entry.Title)[1],
					regexFlightType.FindStringSubmatch(entry.Title)[1],
					entry.Link,
					publicationDate.UnixMilli(),
				}
				flightExists, err := manager.FlightExists(flight)
				if err != nil {
					log.Errorf("Error searching if the flight exists: %v", err)
				}
				if flightExists {
					log.Infof("Flight %v already exists, skipping.", flight)
				} else {
					err = manager.InsertFlight(flight)
					numInsertion++
					if err != nil {
						log.Fatalf("Error indexing flight into ElasticSearch: %v", err)
					}
				}
			}
			log.Infof("Number of flights inserted: %d", len(flights.Channel.Items))
		case <-interrupt:
			log.Info("Exiting.")
			return
		}
	}
}
