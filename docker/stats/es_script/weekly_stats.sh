#!/usr/bin/env sh

# Setup script for elasticsearch. This script MUST be run before any indexing.
# This script can be executed several times on the same elasticsearch cluster (even if no changes will happen after the
# first time).
# The first argument is the URL to the cluster. It must have a HTTP(s) scheme.

if [[ -z "$1" ]] ; then
    echo 'ES URL is missing'
    exit 1
fi


es_cluster_url=$1

destination="/output"
mkdir -p "$destination"

echo "Starting weekly statistics..."

# Set the index template for flights data
cat << EOF | curl -sX POST "$es_cluster_url/flight/_search?pretty=true" -o /output/weekly_stats.json -H "Content-type: application/json" -d @-
{
  "size": 0,
  "aggs": {
    "statistics_yearly": {
      "date_histogram": {
        "field": "flight_date",
        "calendar_interval": "year"
      },
      "aggs": {
        "statistics_weekly": {
          "date_histogram": {
            "field": "flight_date",
            "calendar_interval": "week"
          },
          "aggs": {
            "group_by_country": {
              "terms": {
                "field": "country_code"
              },
              "aggs": {
                "distance_avg": {
                  "avg": {
                    "field": "distance"
                  }
                },
                "distance_med": {
                  "median_absolute_deviation": {
                    "field": "distance"
                  }
                },
                "count_above_50km": {
                  "filter": {
                    "range": { "distance": { "gte": "50" }}

                  }
                },
                "count_above_100km": {
                  "filter": {
                    "range": { "distance": { "gte": "100" }}
                  }
                },
                "count_above_150km": {
                  "filter": {
                    "range": { "distance": { "gte": "150" }}
                  }
                },
                "count_above_200km": {
                  "filter": {
                    "range": { "distance": { "gte": "200" }}
                  }
                }
              }
            }
          }
        }
      }
    }
  }
}
EOF

echo "Weekly statistics computed"
