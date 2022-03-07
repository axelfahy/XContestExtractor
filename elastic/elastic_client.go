package elastic

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"fahy.xyz/xcontestextractor/parser"
	"fmt"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esutil"
	"github.com/mottaquikarim/esquerydsl"
	"github.com/sqooba/go-common/logging"
	"io"
	"io/ioutil"
	"strings"
)

var (
	log = logging.NewLogger()
)

const (
	stateIndexName = "download-state"
)

type ElasticManager struct {
	client    *elasticsearch.Client
	indexName string
}

type SearchResults struct {
	Hits struct {
		Total struct {
			Value int `json:"value"`
		} `json:"total"`
	} `json:"hits"`
}

type LastFlightNumber struct {
	Source struct {
		Year             int `json:"year"`
		LastFlightNumber int `json:"last_flight_number"`
	} `json:"_source"`
}

// NewElasticManager creates a new instance of the ElasticManager.
func NewElasticManager(endpoint string, username string, password string, indexName string) (ElasticManager, error) {
	cfg := elasticsearch.Config{
		Addresses: []string{
			endpoint,
		},
		Username: username,
		Password: password,
	}
	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return ElasticManager{}, err
	}
	client := ElasticManager{
		client:    es,
		indexName: indexName,
	}
	return client, nil
}

// FlightExists check if a flight already exist.
//
// The comparison is done with the full name, the distance and the date of the flight.
func (manager *ElasticManager) FlightExists(fullName string, distance float64, date int64) (bool, error) {
	// Build the request body.
	query, _ := json.Marshal(esquerydsl.QueryDoc{
		Index: manager.indexName,
		And: []esquerydsl.QueryItem{
			{
				Field: "full_name",
				Value: fullName,
				Type:  esquerydsl.Match,
			},
			{
				Field: "distance",
				Value: distance,
				Type:  esquerydsl.Match,
			},
			{
				Field: "flight_date",
				Value: date,
				Type:  esquerydsl.Match,
			},
		},
	})
	log.Debugf("Elasticsearch query: %s", string(query))
	// Perform the search request.
	res, err := manager.client.Search(
		manager.client.Search.WithContext(context.Background()),
		manager.client.Search.WithIndex(manager.indexName),
		manager.client.Search.WithBody(bytes.NewReader(query)),
		manager.client.Search.WithTrackTotalHits(true),
		manager.client.Search.WithPretty(),
	)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()
	if res.StatusCode == 200 {
		var hits SearchResults
		if err = json.NewDecoder(res.Body).Decode(&hits); err != nil {
			return false, nil
		}
		if hits.Hits.Total.Value > 0 {
			return true, nil
		}
	}
	// Read the content of the body before closing.
	_, err = io.Copy(ioutil.Discard, res.Body)
	if err != nil {
		return false, err
	}
	return false, nil
}

// GetStateId compute the hash (id) of a document.
func getStateId(year int) (string, error) {
	h := md5.New()
	if _, err := io.WriteString(h, fmt.Sprintf("%v-%v", stateIndexName, year)); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// GetLastFlightNumber retrieve the number of the last flight processed for the given year.
func (manager *ElasticManager) GetLastFlightNumber(year int) (int, error) {
	// Compute the hash of the document to retrieve.
	hash, err := getStateId(year)
	log.Debugf("Computed hash for year %d: %s", year, hash)
	if err != nil {
		log.Errorf("Unable to compute hash: %v", err)
		return 0, err
	}
	// Search with the correct year.
	res, err := manager.client.Get(stateIndexName, hash)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	log.Debugf("GetLastFlightNumber elasticsearch result: %s", res)
	if res.StatusCode == 200 {
		var flightNumber LastFlightNumber
		if err = json.NewDecoder(res.Body).Decode(&flightNumber); err != nil {
			return 0, err
		}
		log.Debugf("Extracted flight number: %d", flightNumber.Source.LastFlightNumber)
		return flightNumber.Source.LastFlightNumber, nil
	}
	// Read the content of the body before closing.
	if _, err = io.Copy(ioutil.Discard, res.Body); err != nil {
		return 0, err
	}
	log.Warningf("Unable to get last flight number, set to 0.")
	return 0, nil
}

// InsertFlight insert a single flight.
func (manager *ElasticManager) InsertFlight(flight *parser.Flight) error {
	res, err := manager.client.Index(
		manager.indexName,
		esutil.NewJSONReader(flight),
	)
	if err != nil {
		return err
	}
	log.Debugf("InsertFlight elasticsearch result: %s", res)
	return nil
}

// SetLastFlightNumber save the last processed flight number.
func (manager *ElasticManager) SetLastFlightNumber(year int, flightNumber int) error {
	// Compute the hash of the document to save.
	hash, err := getStateId(year)
	log.Debugf("Computed hash for year %d: %s", year, hash)
	if err != nil {
		log.Errorf("Unable to compute hash: %v", err)
		return err
	}
	update := fmt.Sprintf(`{"year": %d, "last_flight_number": %d}`, year, flightNumber)

	res, err := manager.client.Index(
		stateIndexName,
		strings.NewReader(update),
		manager.client.Index.WithDocumentID(hash),
		manager.client.Index.WithRefresh("true"),
	)
	if err != nil {
		return err
	}
	log.Debugf("SetLastFlightNumber elasticsearch result: %s", res)

	return nil
}
