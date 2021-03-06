package main

import (
	"context"
	"flag"
	"fmt"
	"golang.org/x/net/html"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"fahy.xyz/xcontestextractor/elastic"
	"fahy.xyz/xcontestextractor/metrics"
	"fahy.xyz/xcontestextractor/parser"
	browser "github.com/EDDYCJY/fake-useragent"
	"github.com/chromedp/chromedp"
	"github.com/kelseyhightower/envconfig"
	"github.com/sqooba/go-common/logging"
	"github.com/sqooba/go-common/version"
)

const (
	// Index to store the entries.
	indexName     string = "flight"
	source        string = "archive"
	flightsByPage int    = 100
)

var (
	log = logging.NewLogger()
)

type envConfig struct {
	// Logging
	LogLevel string `envconfig:"LOG_LEVEL" default:"debug"`
	// ElasticSearch
	ElasticEndpoint string `envconfig:"ELASTICSEARCH_URL" default:"http://127.0.0.1:9200"`
	ElasticUser     string `envconfig:"ELASTICSEARCH_USERNAME" default:"CHANGEME"`
	ElasticPassword string `envconfig:"ELASTICSEARCH_PASSWORD" default:"CHANGEME"`
	// Prometheus
	MetricsNamespace string `envconfig:"METRICS_NAMESPACE" default:"xcontest"`
	MetricsSubsystem string `envconfig:"METRICS_SUBSYSTEM" default:"archextractor"`
	MetricsPath      string `envconfig:"METRICS_PATH" default:"/metrics"`
	Port             string `envconfig:"PORT" default:"9095"`
	// URL to extract
	Url string `envconfig:"URL"`
	// Start of the extraction (part of the url [start]=)
	StartFlightNumber    int  `envconfig:"START_FLIGHT_NUMBER"` // Only used if `LOAD_LAST_FLIGHT_NUMBER` is false.
	LoadLastFlightNumber bool `envconfig:"LOAD_LAST_FLIGHT_NUMBER" default:"true"`
	// Timeouts and number of retries for chromedp
	TimeoutSeconds  int `envconfig:"TIMEOUT_SECONDS" default:"60"`
	NumberOfRetries int `envconfig:"NUMBER_OF_RETRIES" default:"5"`
	// Interval between run to avoid doing too many requests
	IntervalMin int    `envconfig:"RUN_INTERVAL_MINUTES" default:"2"`
	LogPath     string `envconfig:"LOG_PATH" default:"./logs/archextractor"`
}

// Entry represents a flight.
type Entry struct {
	FullName   string  `json:"full_name"`
	FlightDate int64   `json:"flight_date"`
	Distance   float64 `json:"distance"`
	FlightType string  `json:"flight_type"`
	Link       string  `json:"link"`
}

// getFlights retrieves the files from a html pages.
func getFlights(url string, timeoutSecond int) (string, error) {
	const sel = "html body div#page.sect-cpp div#page-inner div#main-box div.in1 div#content-and-context div#content div.under-bar div#flights.XContest table.XClist tbody"

	opts := []chromedp.ExecAllocatorOption{
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.Headless,
		chromedp.UserAgent(browser.Random()),
		chromedp.DisableGPU,
		chromedp.ExecPath("/headless-shell/headless-shell"),
	}

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	chromeCtx, chromeCancel := chromedp.NewContext(
		allocCtx,
		chromedp.WithLogf(log.Printf),
	)
	defer chromeCancel()

	timeoutCtx, timeoutCancel := context.WithTimeout(chromeCtx, time.Duration(timeoutSecond)*time.Second)
	defer timeoutCancel()

	var res string

	err := chromedp.Run(timeoutCtx,
		chromedp.Navigate(url),
		chromedp.WaitVisible("table.XClist"),
		chromedp.OuterHTML(sel, &res, chromedp.BySearch),
	)
	if err != nil {
		log.Errorf("Error navigating the page: %v", err)
		return "", err
	}
	return res, nil
}

func main() {
	log.Infoln("Starting XContestArchExtractor...")
	log.Infof("Version               : %s", version.Version)
	log.Infof("Commit                : %s", version.GitCommit)
	log.Infof("Build date            : %s", version.BuildDate)
	log.Infof("OSarch                : %s", version.OsArch)

	flag.Parse()

	// Loading env variables.
	var env envConfig
	if err := envconfig.Process("", &env); err != nil {
		log.Fatalf("Failed to process env var: %v", err)
	}
	log.Infof("Elastic endpoint      : %s", env.ElasticEndpoint)
	log.Infof("Elastic user          : %s", env.ElasticUser)

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

	// Initialization of the ElasticSearch client.
	manager, err := elastic.NewElasticManager(
		env.ElasticEndpoint,
		env.ElasticUser,
		env.ElasticPassword,
		indexName,
	)
	if err != nil {
		log.Fatalf("Error creating the ES client: %v", err)
	}

	// Extract year from url.
	re := regexp.MustCompile(`[0-9]{4}`)
	match := re.FindString(env.Url)
	year, err := strconv.Atoi(match)
	if err != nil {
		log.Fatalf("Error extracting the year from the url: %v", err)
	}
	log.Infof("Processing year %d", year)

	// Extract the last flight number processed.
	flightNumber := 0
	if env.LoadLastFlightNumber {
		flightNumber, err = manager.GetLastFlightNumber(year)
		if err != nil {
			log.Fatalf("Error extracting the last flight number: %v", err)
		}
	} else {
		flightNumber = env.StartFlightNumber
	}
	log.Infof("Last flight number: %d", flightNumber)

	retry := 0

	// Process all the pages until there is no more flight.
	for {
		metrics.RunsTotal.Inc()
		url := env.Url + strconv.Itoa(flightNumber)
		log.Infof("Extracting: %s", url)

		data, _ := getFlights(url, env.TimeoutSeconds)
		metrics.HttpRequestsTotal.Inc()
		// If the page is empty, retry ten times before quitting.
		if strings.TrimSpace(data) == "" {
			log.Infof("No more flight to insert (flight number=%d)", flightNumber)
			metrics.ErrorsTotal.Inc()
			if retry < env.NumberOfRetries {
				retry++
				continue
			}
			break
		}
		// Reset the retry counter if we get a non-empty page.
		retry = 0

		var entry Entry

		tokenizer := html.NewTokenizer(strings.NewReader(data))
		// Iterate over all the tags
		for {
			tokenType := tokenizer.Next()

			// If it's an error token, we either reached
			// the end of the file, or the HTML was malformed.
			if tokenType == html.ErrorToken {
				err := tokenizer.Err()
				if err == io.EOF {
					// End of the file, break out of the loop.
					break
				}
				log.Fatalf("Error tokenizing HTML: %v", tokenizer.Err())
			}
			innerToken := tokenizer.Token()

			if tokenType == html.EndTagToken && innerToken.Data == "tr" {
				// Create a new flight.
				log.Debugf("Entry to check: %+v", entry)
				// Check if the flight exists.
				flightExists, err := manager.FlightExists(entry.FullName, entry.Distance, entry.FlightDate)
				if err != nil {
					metrics.ErrorsTotal.Inc()
					log.Errorf("Error searching if the flight exists: %v", err)
				}
				if flightExists {
					log.Info("Flight already exists, skipping.")
					metrics.DuplicatesTotal.Inc()
				} else {
					log.Debugf("Getting flight info of %s at %d (%f km)", entry.FullName, entry.FlightDate, entry.Distance)
					flight, err := parser.GetFlightInfo(entry.Link, source)
					metrics.HttpRequestsTotal.Inc()
					if err != nil {
						metrics.ErrorsTotal.Inc()
						log.Errorf("Error getting flight information of %s: %v", entry.Link, err)
						continue
					}

					flight.FullName = entry.FullName
					flight.FlightDate = entry.FlightDate
					flight.Distance = entry.Distance
					flight.FlightType = entry.FlightType
					//flight.PublicationDate = publicationDate.UnixMilli()
					// TODO: what to put as publication date
					flight.Url = entry.Link

					log.Debugf("Flight to insert: %+v", flight)

					err = manager.InsertFlight(flight)
					log.Debug("Flight inserted successfully.")
					metrics.DocumentsTotal.Inc()
					if err != nil {
						metrics.ErrorsTotal.Inc()
						log.Fatalf("Error indexing flight into ElasticSearch: %v", err)
					}
				}
			}
			if tokenType == html.StartTagToken {
				switch data := innerToken.Data; data {
				// Create a new structure flight.
				case "tr":
					log.Debug("Creating a new flight structure.")
					entry = Entry{}
				// Extract the full name.
				case "b":
					tokenizer.Next()
					entry.FullName = string(tokenizer.Text())
				// Extract the link of the flight.
				case "a":
					if len(innerToken.Attr) > 0 && innerToken.Attr[0].Val == "detail" {
						entry.Link = innerToken.Attr[2].Val
						log.Debugf("Extracted link: %s", entry.Link)
						// Extract the date.
						split := strings.Split(entry.Link, "/")
						date, err := parser.ParseDate(split[len(split)-2])
						if err != nil {
							metrics.ErrorsTotal.Inc()
							log.Errorf("Error converting the date flight of %s: %v", entry.Link, err)
							continue
						}
						entry.FlightDate = date.UnixMilli()
					}
				// Extract the type of flight.
				case "div":
					if len(innerToken.Attr) > 0 && strings.Contains(innerToken.Attr[0].Val, "disc") {
						entry.FlightType = strings.ToLower(strings.Replace(innerToken.Attr[1].Val, " ", "_", -1))
					}
				// Extract the distance.
				case "td":
					if len(innerToken.Attr) > 0 && innerToken.Attr[0].Val == "km" {
						tokenizer.Next()
						tokenizer.Next() // Skip the <strong>.
						distance, err := strconv.ParseFloat(string(tokenizer.Text()), 64)
						if err != nil {
							metrics.ErrorsTotal.Inc()
							log.Errorf("Error converting distance flight to float of %s: %v", entry.Link, err)
							continue
						}
						entry.Distance = distance
					}
				}
			}
		}
		flightNumber += flightsByPage
		if err = manager.SetLastFlightNumber(year, flightNumber); err != nil {
			metrics.ErrorsTotal.Inc()
			log.Errorf("Error while setting the last flight number: %v", err)
		}
		time.Sleep(time.Duration(env.IntervalMin) * time.Minute)
	}

	log.Info("Flights successfully imported.")
	time.Sleep(30 * time.Second)
}
