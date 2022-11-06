# -*- coding: utf-8 -*-
"""Transformer for XContest statistics."""
from datetime import datetime, timezone
import json

import click
from loguru import logger
from pathlib import Path


def extract_countries(data: dict) -> list:
    """
    Extract complete list of countries from data.

    Parameters
    ----------
    data : dict
        Dictionnary with all the data.

    Returns
    -------
    list
        List of unique countries, sorted.
    """
    countries = set()
    for years in data["aggregations"]["statistics_yearly"]["buckets"]:
        for weeks in years["statistics_weekly"]["buckets"]:
            for country in weeks["group_by_country"]["buckets"]:
                countries.add(country["key"])
    return sorted(countries)


@click.command()
@click.argument('filename', type=click.Path(exists=True))
@click.argument('output', type=click.Path(exists=False))
def main(filename: Path, output: Path):
    """
    Transform the input file for statistics.

    Creates a json file group by year, week number, and stats.
    """
    logger.info(f"Starting transformer on {filename}")
    with Path(filename).open() as f:
        raw_data = json.load(f)

        data = {}
        for years in raw_data["aggregations"]["statistics_yearly"]["buckets"]:
            year = datetime.fromtimestamp(years['key'] // 1000, timezone.utc).year
            data[year] = {c: {} for c in extract_countries(raw_data)}
            for weeks in years["statistics_weekly"]["buckets"]:
                week_number = datetime.fromtimestamp(weeks['key'] // 1000, timezone.utc).isocalendar()[1]
                for country in weeks["group_by_country"]["buckets"]:
                    data[year][country["key"]][week_number] = {
                        "distance_avg": country["distance_avg"]["value"],
                        "distance_med": country["distance_med"]["value"],
                        "count_above_50km": country["count_above_50km"]["doc_count"],
                        "count_above_100km": country["count_above_100km"]["doc_count"],
                        "count_above_150km": country["count_above_150km"]["doc_count"],
                        "count_above_200km": country["count_above_200km"]["doc_count"],
                    }

    with Path(output).open(mode='w') as f:
        json.dump(data, f)

    logger.info(f"Output saved at {output}")

if __name__ == '__main__':
    main() # pylint: disable=no-value-for-parameter
