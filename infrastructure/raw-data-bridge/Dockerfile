# SPDX-FileCopyrightText: 2024 NOI Techpark <digital@noi.bz.it>
#
# SPDX-License-Identifier: CC0-1.0

FROM golang:1.23.7-bookworm AS base

FROM base AS build
WORKDIR /app
COPY go.mod .
COPY go.sum .
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o raw-data-bridge

FROM scratch AS release
COPY --from=build /app/raw-data-bridge /raw-data-bridge
CMD ["/raw-data-bridge"]


FROM base AS dev
WORKDIR /code
CMD ["go", "run", "."]
