FROM alpine:latest

RUN apk add --no-cache curl

COPY dev/docker/setup_indexing/es_script /opt/es_script/

RUN ln -s /opt/es_script/docker-entrypoint.sh

ENTRYPOINT ["./docker-entrypoint.sh"]
