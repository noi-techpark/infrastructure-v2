# Start Local Cluster

1. Install [Kind](https://kind.sigs.k8s.io/) (Kubernetes in Docker)
2. Start cluster
```
kind create cluster --name open-data-hub --config local-cluster.yaml
```

By default the cluster will restart every time you reboot the PC. If you want to prevent it and only start it when needed:

```sh
for container in $(docker ps --filter "name=open-data-hub-" --format "{{.Names}}"); do
  docker update --restart=no "$container"
done
```

## Restart the cluster after a reboot

```
docker ps --all | grep open-data-hub- | awk '{print $1}' | xargs docker restart
```

# Deploy Prometheus

1.
```
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
```

2.
```
helm upgrade --install prometheus prometheus-community/kube-prometheus-stack -f infrastructure/helm/prometheus/values.yaml -n monitoring --create-namespace
```

2. When deploying in **KIND**
```
helm upgrade --install prometheus prometheus-community/kube-prometheus-stack \
  -f infrastructure/helm/prometheus/values.yaml \
  --set prometheus.prometheusSpec.storageSpec.volumeClaimTemplate.spec.storageClassName=standard \
  --set alertmanager.alertmanagerSpec.storage.volumeClaimTemplate.spec.storageClassName=standard \
  -n monitoring --create-namespace
```

## Thanos
Thanos is an extra layer between metrics consumer (like grafana) and prometheus.
Thanos is useful to handle multiple instances of prometheus for High Availability and for long time storage. It requires sidecar and the whole Thanos stack to be deployed.

Installation reference [here](https://medium.com/@dast04/full-installation-monitoring-with-prometheus-thanos-part-2-2-fe98fcdbe448).

For now we won't use HA prometheus.

## Pods Configuration

Service Discovery: Prometheus uses Kubernetes service discovery to find the pods and their exposed endpoints.
Annotations: You can add annotations to your pod or service definitions to specify the metrics endpoint. For example:

```
metadata:
    annotations:
    prometheus.io/scrape: "true"
    prometheus.io/path: "/metrics"
    prometheus.io/port: "8080"
```

## Service/Pod Monitoring

**NOT MORE TRUE, JUST NEED TO ADD serviceMonitorSelectorNilUsesHelmValues and podMonitorSelectorNilUsesHelmValues to prometheusSpec**

A ServiceMonitor need to have a particular label to be scraped by Prometheus Operator:
```yaml
metadata:
  name: <name>

  labels:
    release: prometheus # < This label
....
```

[Reference](https://managedkube.com/prometheus/operator/servicemonitor/troubleshooting/2019/11/07/prometheus-operator-servicemonitor-troubleshooting.html)

## Service/pod Monitor
Service and pod monitor resources tells prometheus what to scrape how to scrape

### Rabbimq
** NOT NEEDED ANYMORE, SERVICEMONITOR ALREADY IN RABBIT'S VALUES.YAML **

Deploy RabbitMQ ServiceMonitor

```
kubectl apply --filename infrastructure/helm/rabbitmq/service-monitor.yaml -n monitoring
```

Scraped metrics are described here: [Monitoring RabbitMQ](https://www.rabbitmq.com/docs/prometheus#detailed-endpoint)

## Mongodb exporter

The bellow manifest creates mongdb-exporter pod, service and serviceMonitor


```
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

helm upgrade --install mongodb-prometheus-exporter prometheus-community/prometheus-mongodb-exporter -f infrastructure/helm/mongodb/exporter.yaml -n monitoring
```

MongoDB exporter has a field in the values to control the instance name
```
  --set serviceMonitor.metricRelabelings[0].targetLabel="instance" \
  --set serviceMonitor.metricRelabelings[0].replacement="...."
```

[Exporter image Dockerhub](https://hub.docker.com/r/percona/mongodb_exporter)
[Exporter Repository](https://github.com/percona/mongodb_exporter)
[Exporter Helm Chart](https://artifacthub.io/packages/helm/prometheus-community/prometheus-mongodb-exporter)

# Deploy Tempo

We will use `tempo-distributed`.

1.
```
helm repo add grafana https://grafana.github.io/helm-charts
```

2.
```
helm upgrade --install tempo grafana/tempo-distributed -f infrastructure/helm/tempo/values.yaml -n monitoring
```

2. When deploying in **KIND**
```
helm upgrade --install tempo grafana/tempo-distributed -f infrastructure/helm/tempo/values.yaml \
  --set ingester.persistence.enabled=false \
  -n monitoring --create-namespace
```