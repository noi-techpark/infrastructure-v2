# SPDX-FileCopyrightText: 2024 NOI Techpark <digital@noi.bz.it>
#
# SPDX-License-Identifier: CC0-1.0

FROM golang:1.22.2-bookworm AS base

FROM base AS build
WORKDIR app
COPY src/main main
CMD [ "./main"]

FROM base AS dev
WORKDIR /code

CMD ["go", "run", "main.go"]
