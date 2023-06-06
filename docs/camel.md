<!--
SPDX-FileCopyrightText: NOI Techpark <digital@noi.bz.it>

SPDX-License-Identifier: CC0-1.0
-->

# Camel

## Prerequisites

In order to run, compiled, and debug Camel routes, it is necessary to satisfy the following prerequisites:

- Installation of [Kamel CLI](https://camel.apache.org/camel-k/1.9.x/cli/cli.html) (1.9.x)
- Prerequisites of [Kubernetes](kubernetes.md#Prerequisites)

## Known Limitations

**Amazon ECR is not supported yet by Kamel, use [Docker Hub](https://hub.docker.com/) instead.**

## Exchange Pattern

There are two *Message Exchange Patterns* you can use in messaging.

According to the Enterprise Integration Patterns, they are:

- Event Message (or one-way)
- Request-Reply

In Camel, we have an org.apache.camel.ExchangePattern enumeration which can be configured on the exchangePattern property on the Message Exchange indicating if a message exchange is a one-way Event Message (**InOnly**) or a Request-Reply message exchange (**InOut**).

More information in the [official documentation](https://camel.apache.org/manual/exchange-pattern.html).

## Camel K

When deployed in Kubernetes the Camel routes are deployed using Camel K.
Camel K does not deploy the whole Maven/Quarkus/Spring-boot project but only the route definition. To do so it needs a special syntax in the .java files to know which package to install.

Example:
```java
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-bean
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-seda
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-stream
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-paho-mqtt5
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-openapi-java

package it.bz.opendatahub.inbound.mqtt;

import org.apache.camel.builder.RouteBuilder;
import org.apache.camel.component.paho.mqtt5.PahoMqtt5Constants;
```

It's also recommended to keep the route in a single file.
Reference: [Blog Article](https://piotrminkowski.com/2020/12/08/apache-camel-k-and-quarkus-on-kubernetes/).

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
It's not possible to scale down the Kamel integration from Kubernetes.
If we need to stop a route we have to tell Kamel to delete the whole route, which reflects in deleting the Kubernetes deployment.

```
kamel delete <route-name>
```