# /bin/sh

docker build -t kind/infrastructure-v2/raw-data-bridge:v1.0 --target release -f infrastructure/raw-data-bridge/Dockerfile ~/
docker build -t kind/infrastructure-v2/raw-writer:v1.0 --target release -f infrastructure/raw-writer/Dockerfile ~/
docker build -t kind/infrastructure-v2/router:v1.0 --target release -f infrastructure/router/Dockerfile ~/

kind load docker-image kind/infrastructure-v2/raw-data-bridge:v1.0 --name rabbitmq-cluster
kind load docker-image kind/infrastructure-v2/raw-writer:v1.0 --name rabbitmq-cluster
kind load docker-image kind/infrastructure-v2/router:v1.0 --name rabbitmq-cluster

helm upgrade --install -n core raw-data-bridge ./infrastructure/helm/raw-data-bridge/raw-data-bridge -f ./infrastructure/helm/raw-data-bridge/values.yaml
helm upgrade --install -n core raw-writer ./infrastructure/helm/raw-writer/raw-writer -f ./infrastructure/helm/raw-writer/values.yaml
helm upgrade --install -n core router ./infrastructure/helm/router/router -f ./infrastructure/helm/router/values.yaml
