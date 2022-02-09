FROM alpine:latest

RUN apk add --no-cache bash curl

COPY docker/stats/stats-crontab /etc/cron.d/stats-crontab
RUN chmod 0644 /etc/cron.d/stats-crontab && \
    crontab /etc/cron.d/stats-crontab
COPY docker/stats/es_script /opt/es_script/

ENTRYPOINT ["crond", "-f"]
