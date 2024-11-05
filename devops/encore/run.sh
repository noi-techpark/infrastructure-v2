#!/bin/bash

# Port-forward services
kubectl port-forward -n core svc/rabbitmq-headless 3001:5672 --address 0.0.0.0 &
forwards=$!
kubectl port-forward -n core svc/mongodb-headless 3002:27017 --address 0.0.0.0 &
forwards="$forwards $!"

cleanup()
{
for pid in $forwards; do
    kill $pid
done
}

trap cleanup EXIT 

# Connection details for rabbit and mongo. Run kubectl port-forward above beforehand
RABBIT_PW=`kubectl get secret -n collector rabbitmq-svcbind -o  jsonpath='{.data.password}' | base64 -d`
MONGO_PW=`kubectl get secret -n collector mongodb-collector-svcbind -o  jsonpath='{.data.password}' | base64 -d`
export RABBIT_URI="amqp://opendatahub:${RABBIT_PW}@localhost:3001"
export MONGO_URI="mongodb://collector:${MONGO_PW}@localhost:3002/?tls=false&ssl=false&directConnection=true"

if [ ! -f .env ]; then
    echo ".env file does not exist. Copy and customize the example file"
    exit 1
fi
source .env

go mod download
go run main.go