#!/bin/bash
IMAGE=ghcr.io/noi-techpark/infrastructure-v2/raw-writer-2
TAG=${IMAGE_TAG:-$(git rev-parse HEAD)}
REPO=https://github.com/noi-techpark/infrastructure-v2
docker buildx build --target release -f Dockerfile . -t $IMAGE \
--label "org.opencontainers.image.source=$REPO" \
--label "org.opencontainers.image.description=Raw data writer service for Open Data Hub ingestion pipeline" \
--label "org.opencontainers.image.licenses=AGPL-3.0-or-later"

docker tag $IMAGE "$IMAGE:$TAG"
docker tag $IMAGE "$IMAGE:latest"

docker push --all-tags $IMAGE