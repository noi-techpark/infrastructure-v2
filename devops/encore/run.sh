#!/bin/bash

# Port-forward services
kubectl port-forward -n core svc/rabbitmq-headless 5672 --address 0.0.0.0 &
forwards=$!
kubectl port-forward -n core svc/mongodb-headless 27017 --address 0.0.0.0 &
forwards="$forwards $!"

# Connection details for rabbit and mongo. Run kubectl port-forward above beforehand
RABBIT_PW=`kubectl get secret -n collector rabbitmq-svcbind -o  jsonpath='{.data.password}' | base64 -d`
MONGO_PW=`kubectl get secret -n collector mongodb-collector-svcbind -o  jsonpath='{.data.password}' | base64 -d`
export RABBIT_URI="amqp://opendatahub:${RABBIT_PW}@localhost:5672"
export MONGO_URI="mongodb://collector:${MONGO_PW}@localhost/?tls=false&ssl=false&directConnection=true"

# Mongodb db and collection that is queried
export DB="municipality-bolzano"
export COLLECTION="parking-macello"

# Queue the messages are pushed into
export QUEUE="municipality-bolzano.parking-macello.temp"

# A mongodb query that is applied to the collection
# Following the "relaxed" format of mongodb extended json: https://www.mongodb.com/docs/manual/reference/mongodb-extended-json/
# e.g. Dates have to be full ISO not just partial
export QUERY='{ "bsontimestamp": { "$gte": { "$date": "2024-06-21T00:00:00.000+00:00"}}}'

go mod download
go run main.go

for pid in $forwards; do
    kill $pid
done