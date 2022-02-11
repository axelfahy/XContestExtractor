#!/usr/bin/env sh

/opt/script/weekly_stats.sh $ES_CLUSTER_URL
python /opt/script/transformer.py /output/weekly_stats.raw.json /output/weekly_stats.processed.json
