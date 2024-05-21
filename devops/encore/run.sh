#!/bin/bash
#
# In another terminal, start
# kubectl port-forward -n core svc/rabbitmq-headless 5672 --address 0.0.0.0 &
# kubectl port-forward -n core svc/rabbitmq-headless 15672 --address 0.0.0.0 &
# kubectl port-forward -n core svc/mongodb-headless 27017 --address 0.0.0.0 &
#
# And then kill them with
# killall kubectl


set -o allexport
source .env
set +o allexport
cd src
go run main.go
