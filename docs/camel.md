# Camel

## Prerequisites

In order to run, compiled, and debug Camel routes, it is necessary to satisfy the following prerequisites:

- Installation of [Kamel CLI](https://camel.apache.org/camel-k/1.9.x/cli/cli.html) (1.9.x)
- Prerequisites of [Kubernetes](kubernetes.md#Prerequisites)

## Known Limitations

**Amazon ECR is not supported yet by Kamel, use [Docker Hub](https://hub.docker.com/) instead.**

## How to

### Run Camel Routes

```
kamel run \
  --name mqtt-route \
  --property mqtt.url=tcp://mosquitto-edge:1883 \
  --property internal_mqtt.url=tcp://mosquitto-storage.default.svc.cluster.local:1883 \
  --property internal_mqtt.topic=storage \
    infrastructure/inbound/src/main/java/it/bz/opendatahub/inbound/mqtt/MqttRoute.java
```

```
kamel run \
  --name rest-route \
  --property internal_mqtt.url=tcp://mosquitto-storage.default.svc.cluster.local:1883 \
  --property internal_mqtt.topic=storage \
    infrastructure/inbound/src/main/java/it/bz/opendatahub/inbound/rest/RestRoute.java
```

```
kamel run \
  --name writer-route \
  --property quarkus.mongodb.connection-string=mongodb://mongodb-0.mongodb-headless.default.svc.cluster.local:27017 \
  --property quarkus.mongodb.devservices.enabled=false \
  --property internal_mqtt.url=tcp://mosquitto-storage.default.svc.cluster.local:1883 \
  --property internal_mqtt.topic=storage \
    infrastructure/writer/src/main/java/it/bz/opendatahub/writer/WriterRoute.java
```

### Delete Camel Routes
It's not possible to scale down the kamel integration from kubernetes.
If we need to stop a route w have to tell kamel to delete the whole route, which reflects in deleting the kubernetes deployment.

```
kamel delete <route-name>
```