package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"time"

	"fahy.xyz/xcontestextractor/elastic"
	"fahy.xyz/xcontestextractor/metrics"
	"fahy.xyz/xcontestextractor/parser"
	"github.com/kelseyhightower/envconfig"
	"github.com/sqooba/go-common/logging"
	"github.com/sqooba/go-common/version"
)

const (
	// Index to store the entries.
	indexName string = "flight-000001"
	source    string = "rss"
	// Url of the RSS feed of XContest.
	url string = "https://www.xcontest.org/rss/flights/?world"
	// Date formats.
	pubDateLayout    string = "Mon, 2 Jan 2006 15:04:05 +0000"
	flightDateLayout string = "02.01.06"
)

var (
	log = logging.NewLogger()
)

type envConfig struct {
	// Logging
	LogLevel string `envconfig:"LOG_LEVEL" default:"info"`
	// ElasticSearch
	ElasticEndpoint string `envconfig:"ELASTICSEARCH_URL" default:"http://127.0.0.1:9200"`
	ElasticUser     string `envconfig:"ELASTICSEARCH_USERNAME" default:"CHANGEME"`
	ElasticPassword string `envconfig:"ELASTICSEARCH_PASSWORD" default:"CHANGEME"`
	// Prometheus
	MetricsNamespace string `envconfig:"METRICS_NAMESPACE" default:"xcontest"`
	MetricsSubsystem string `envconfig:"METRICS_SUBSYSTEM" default:"rssextractor"`
	MetricsPath      string `envconfig:"METRICS_PATH" default:"/metrics"`
	Port             string `envconfig:"PORT" default:"9095"`
	// App
	IntervalMin int `envconfig:"RUN_INTERVAL_MINUTES" default:"5"`
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
	log.Infoln("Starting XContestRSSExtractor...")
	log.Infof("Version               : %s", version.Version)
	log.Infof("Commit                : %s", version.GitCommit)
	log.Infof("Build date            : %s", version.BuildDate)
	log.Infof("OSarch                : %s", version.OsArch)

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
	log.Infof("Elastic endpoint      : %s", env.ElasticEndpoint)
	log.Infof("Elastic user          : %s", env.ElasticUser)
	log.Infof("Running interval [m]  : %d", env.IntervalMin)

	if err := logging.SetLogLevel(log, env.LogLevel); err != nil {
		log.Fatalf("Logging level %s do not seem to be right, err = %v", env.LogLevel, err)
	}

	// Start prometheus server.
	mConfig := metrics.Config{
		Namespace: env.MetricsNamespace,
		Subsystem: env.MetricsSubsystem,
		Path:      env.MetricsPath,
	}
	metrics.InitPrometheus(mConfig, http.DefaultServeMux)
	s := http.Server{Addr: fmt.Sprint(":", env.Port)}
	go func() {
		log.Fatal(s.ListenAndServe())
	}()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	ticker := time.NewTicker(time.Duration(env.IntervalMin) * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case t := <-ticker.C:
			log.Infof("Running extractor at %v", t)
			metrics.RunsTotal.Inc()

			// Initialization of the ElasticSearch client.
			manager := elastic.NewElasticManager(
				env.ElasticEndpoint,
				env.ElasticUser,
				env.ElasticPassword,
				indexName,
			)

			// Read the RSS feed.
			resp, err := http.Get(url)
			metrics.HttpRequestsTotal.Inc()
			if err != nil {
				metrics.ErrorsTotal.Inc()
				log.Errorf("Error requesting url: %v", err)
				continue
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
				log.Debugf("Processing flight  : %s (%d / %d)", entry, i, len(flights.Channel.Items))

				fullName := regexFullName.FindStringSubmatch(entry.Title)[1]
				log.Debugf("Full name          : %s", fullName)
				distance, err := strconv.ParseFloat(regexDistance.FindStringSubmatch(entry.Title)[1], 64)
				if err != nil {
					metrics.ErrorsTotal.Inc()
					log.Fatalf("Error converting distance flight to float: %v", err)
				}
				log.Debugf("Distance           : %f", distance)

				date, err := time.Parse(flightDateLayout, strings.Split(entry.Title, " ")[0])
				if err != nil {
					metrics.ErrorsTotal.Inc()
					log.Fatalf("Error converting date flight to timestamp: %v", err)
				}
				log.Debugf("Date               : %s", date)

				flightExists, err := manager.FlightExists(fullName, distance, date.UnixMilli())
				if err != nil {
					metrics.ErrorsTotal.Inc()
					log.Errorf("Error searching if the flight exists: %v", err)
				}
				if flightExists {
					log.Info("Flight already exists, skipping.")
					metrics.DuplicatesTotal.Inc()
				} else {
					flight, err := parser.GetFlightInfo(entry.Link, source)
					if err != nil {
						metrics.ErrorsTotal.Inc()
						log.Errorf("Error getting flight information: %v", err)
					}
					publicationDate, err := time.Parse(pubDateLayout, entry.PubDate)
					if err != nil {
						metrics.ErrorsTotal.Inc()
						log.Fatalf("Error converting publication date to timestamp: %v", err)
					}
					log.Debugf("Publication date   : %s", publicationDate)

					flight.FullName = fullName
					flight.FlightDate = date.UnixMilli()
					flight.Distance = distance
					flight.FlightType = regexFlightType.FindStringSubmatch(entry.Title)[1]
					flight.PublicationDate = publicationDate.UnixMilli()
					flight.Url = entry.Link

					log.Debugf("Flight type        : %s", flight.FlightType)
					log.Debugf("Url                : %s", flight.Url)

					err = manager.InsertFlight(flight)
					metrics.DocumentsTotal.Inc()
					numInsertion++
					if err != nil {
						log.Fatalf("Error indexing flight into ElasticSearch: %v", err)
					}
				}
			}
			log.Infof("Number of flights inserted: %d", numInsertion)
		case <-interrupt:
			log.Info("Exiting.")
			return
		}
	}
}
