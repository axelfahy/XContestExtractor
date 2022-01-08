package elastic

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esutil"
	"github.com/mottaquikarim/esquerydsl"
	"io"
	"io/ioutil"

	"fahy.xyz/xcontestextractor/parser"
	"github.com/sqooba/go-common/logging"
)

var (
	log = logging.NewLogger()
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

// NewElasticManager creates a new instance of the ElasticManager.
func NewElasticManager(endpoint string, username string, password string, indexName string) ElasticManager {
	cfg := elasticsearch.Config{
		Addresses: []string{
			endpoint,
		},
		Username: username,
		Password: password,
	}
	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		log.Errorf("Error creating the ES client: %v", err)
	}
	client := ElasticManager{
		client:    es,
		indexName: indexName,
	}
	return client
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
	log.Debugf("Elasticsearch result: %s", res)
	return nil
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
