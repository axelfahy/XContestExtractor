# XContest Extractor

- Extract flights from the RSS feed of XContest (https://www.xcontest.org/rss/flights/?world).

- Extract flights from the archive.

## RSS Extractor

1. Extract data from the RSS feed of XContest.
2. Get more information about the flight using its url.
3. Insert flights into ElasticSearch if they don't exist.

## Archive Extractor

1. Download a page from the daily-score.
2. Parse the tokens to get all the flights information.
3. Insert flights into ElasticSearch if they don't exist.

## Execution

The tools are run using docker.

