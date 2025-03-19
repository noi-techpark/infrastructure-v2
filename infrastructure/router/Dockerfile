# SPDX-FileCopyrightText: 2024 NOI Techpark <digital@noi.bz.it>
#
# SPDX-License-Identifier: CC0-1.0

FROM golang:1.23.4-bookworm AS base

FROM base AS build
WORKDIR /app
COPY git/infrastructure-v2/infrastructure/router/go.mod git/infrastructure-v2/infrastructure/router/go.mod
COPY git/infrastructure-v2/infrastructure/router/go.sum git/infrastructure-v2/infrastructure/router/go.sum
COPY git/infrastructure-v2/infrastructure/router/ git/infrastructure-v2/infrastructure/router

COPY exp/k8s/x/ exp/k8s/x

RUN cd git/infrastructure-v2/infrastructure/router && CGO_ENABLED=0 GOOS=linux go build -o router

FROM scratch AS release
COPY --from=build /app/git/infrastructure-v2/infrastructure/router/router /router
CMD ["/router"]

FROM base AS dev
WORKDIR /code
CMD ["go", "run", "."]
