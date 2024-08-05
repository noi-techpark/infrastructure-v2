#!/bin/bash
IMAGE=ghcr.io/noi-techpark/infrastructure-v2/raw-writer
REPO=https://github.com/noi-techpark/infrastructure-v2
docker buildx build --target release -f Dockerfile . \
-t $IMAGE \
-t $IMAGE $IMAGE:latest \
-t $IMAGE $IMAGE:v1 \
-t $IMAGE $IMAGE:v1.0 \
--label "org.opencontainers.image.source=$REPO" \
--label "org.opencontainers.image.description=Raw data writer service for Open Data Hub ingestion pipeline" \
--label "org.opencontainers.image.licenses=AGPL-3.0-or-later"

docker push --all-tags $IMAGE