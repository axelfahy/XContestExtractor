#!/usr/bin/env sh

# Setup script for elasticsearch. This script MUST be run before any indexing.
# This script can be executed several times on the same elasticsearch cluster
# (even if no changes will happen after the first time).
# The first argument is the URL to the cluster. It must have a HTTP(s) scheme.
# The second argument is the maximal size for an index before a rollover (default: 10GB). The unit is mandatory.

if [[ -z "$1" ]] ; then
    echo 'ES URL is missing'
    exit 1
fi

# Exit when any command fails.
set -e

es_cluster_url=$1
max_size=${2:-10GB}

check_execution()
{
  name=$1
  result=$2
  echo
  if [[ $result == 0 ]] ; then
    echo "Request for ($name) successfully set"
  else
    echo "Request failed ($name)"
  fi
}

echo "Creating index templates..."

template="flight"
echo "Add component template with the mappings of the fields for ${template}"
cat << EOF | curl -sX PUT "${es_cluster_url}/_component_template/${template}-mappings" -H "Content-type: application/json" -d @-
{
  "template": {
    "settings": {
      "number_of_shards": 1
    },
    "mappings": {
      "properties": {
        "flight_date": {
          "type": "date",
          "format": "epoch_millis"
        },
        "full_name": {
          "type": "text"
        },
        "flight_type": {
          "type": "keyword",
          "ignore_above": 256
        },
        "distance": {
          "type": "float"
        },
        "url": {
          "type": "text"
        },
        "publication_date": {
          "type": "date",
          "format": "epoch_millis"
        },
        "average_speed": {
          "type": "float"
        },
        "country_code": {
          "type": "keyword"
        },
        "flight_time": {
          "type": "text"
        },
        "take_off": {
          "type": "keyword"
        },
        "altitude_max": {
          "type": "integer"
        },
        "parsing_source": {
          "type": "keyword"
        }
      }
    }
  },
  "_meta": {
    "description": "To automatically add mappings to indexes with alias $template"
  }
}
EOF
check_execution "${template}-mappings" $?

echo "Add lifecycle policy for ${template} with a maximal size of ${max_size}"
cat << EOF | curl -sX PUT "${es_cluster_url}/_ilm/policy/${template}-ilm-policy" -H "Content-type: application/json" -d @-
{
  "policy": {
    "phases": {
      "hot": {
        "actions": {
          "rollover": {
            "max_primary_shard_size": "$max_size"
          },
          "set_priority": {
            "priority": 100
          }
        }
      }
    }
  }
}
EOF
check_execution "${template}-ilm-policy" $?

echo "Add component template with lifecycle policy for ${template}"
cat << EOF | curl -sX PUT "${es_cluster_url}/_component_template/${template}-ilm-settings" -H "Content-type: application/json" -d @-
{
  "template": {
    "settings": {
      "index.lifecycle.name": "${template}-ilm-policy",
      "index.lifecycle.rollover_alias": "$template"
    }
  },
  "_meta": {
    "description": "To automatically add ilm policy to indexes with alias $template"
  }
}
EOF
check_execution "${template}-ilm-settings" $?

echo "Add index template with previously created component templates"
cat << EOF | curl -sX PUT "${es_cluster_url}/_index_template/${template}" -H "Content-type: application/json" -d @-
{
  "index_patterns": [
    "${template}*"
  ],
  "template": {
    "aliases": {
      "$template": {}
    }
  },
  "composed_of": ["${template}-mappings", "${template}-ilm-settings"],
  "priority": 100,
  "_meta": {
    "description": "To generate expected configuration for ${template} indexes"
  }
}
EOF
check_execution $template $?

if [[ $(curl -s -o /dev/null -w "%{http_code}" "${es_cluster_url}/${template}-000001") -eq 404 ]]; then
  echo "Add first index as write index with the correct alias"
  cat << EOF | curl -sX PUT "${es_cluster_url}/${template}-000001" -H "Content-type: application/json" -d @-
{
  "aliases": {
    "$template": {
      "is_write_index": true
    }
  }
}
EOF
  check_execution "${template}-00001" $?
else
  echo "Index ${template}-000001 already exists, skipping."
fi

echo "Add index template to store the states"
download_template="download-state"
cat << EOF | curl -sX PUT "${es_cluster_url}/_index_template/${download_template}" -H "Content-type: application/json" -d @-
{
  "index_patterns": [
    "download-state*"
  ],
  "template": {
    "settings": {
      "number_of_shards": 1
    },
    "mappings": {
      "properties": {
        "year": {
          "type": "integer"
        },
        "last_flight_number": {
          "type": "integer"
        }
      }
    }
  }
}
EOF
check_execution "${download_template}" $?

if [[ $(curl -s -o /dev/null -w "%{http_code}" "${es_cluster_url}/${download_template}") -eq 404 ]]; then
  echo "Create index ${download_template}"
  curl -sX PUT "$es_cluster_url/${download_template}"
  check_execution "${download_template}" $?
else
  echo "Index ${download_template} already exists, skipping."
fi
