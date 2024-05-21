FROM golang:1.22.3-alpine as build-env

WORKDIR /app
COPY src/* .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o main

# BUILD published image
FROM alpine:latest
WORKDIR /app
COPY --from=build-env /app/main .
ENTRYPOINT [ "./main"]
