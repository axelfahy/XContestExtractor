FROM python:3.10-alpine

RUN apk add --no-cache bash curl

# Crontab
COPY docker/stats/stats-crontab /etc/cron.d/stats-crontab
RUN chmod 0644 /etc/cron.d/stats-crontab && \
    crontab /etc/cron.d/stats-crontab

COPY docker/stats/script /opt/script/

WORKDIR /opt/script

RUN pip install -r requirements.txt

ENTRYPOINT ["crond", "-f"]
