# SPDX-FileCopyrightText: NOI Techpark <digital@noi.bz.it>
#
# SPDX-License-Identifier: CC0-1.0

#!/bin/bash
IMAGE=ghcr.io/noi-techpark/infrastructure-v2/raw-data-bridge
TAGS='latest v1 v1.0'
REPO=https://github.com/noi-techpark/infrastructure-v2
docker buildx build --target release -f Dockerfile . -t $IMAGE \
--label "org.opencontainers.image.source=$REPO" \
--label "org.opencontainers.image.description=Resource Lookup service to retrieve data from Raw Data Table" \
--label "org.opencontainers.image.licenses=AGPL-3.0-or-later"

for tag in $TAGS
do
    docker tag $IMAGE "$IMAGE:$tag"
done

docker push --all-tags $IMAGE