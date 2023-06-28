<!--
SPDX-FileCopyrightText: NOI Techpark <digital@noi.bz.it>

SPDX-License-Identifier: CC0-1.0
-->

# Camel

## Prerequisites

In order to run, compiled, and debug Camel routes, it is necessary to satisfy the following prerequisites:

- Installation of [Kamel CLI](https://github.com/apache/camel-k/releases) (1.12+)
- Prerequisites of [Kubernetes](kubernetes.md#Prerequisites)

## Known Limitations

- **Amazon ECR is not supported yet by Kamel, use [Docker Hub](https://hub.docker.com/) instead.**
- **Only one Java source file is supported. This limitation can be bypassed by building dependencies locally as `.jar`.**
  - Example: `kamel run -d file://path/to/dependency.jar Route.java`

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

## How to deploy routes via Camel K CLI

### Run the **MQTT Route**

```
kamel run \
  --name mqtt-route \
  --property mqtt.url='tcp://mosquitto:1883' \
  --property rabbitmq.cluster='rabbitmq-0.rabbitmq-headless.default.svc.cluster.local:5672' \
  --property rabbitmq.user='guest' \
  --property rabbitmq.pass='guest' \
    infrastructure/inbound/src/main/java/it/bz/opendatahub/inbound/mqtt/MqttRoute.java
```

### Run the **REST Route (REST and WebSocket)**

```
kamel run \
  --name rest-route \
  --property rabbitmq.cluster='rabbitmq-0.rabbitmq-headless.default.svc.cluster.local:5672' \
  --property rabbitmq.user='guest' \
  --property rabbitmq.pass='guest' \
    infrastructure/inbound/src/main/java/it/bz/opendatahub/inbound/rest/RestRoute.java
```

### Run the **Writer Route (MongoDB)**

```
kamel run \
  --name writer-route \
  --property quarkus.mongodb.connection-string='mongodb://mongodb-0.mongodb-headless.default.svc.cluster.local:27017' \
  --property quarkus.mongodb.devservices.enabled=false \
  --property mongodb.host='mongodb-0.mongodb-headless.default.svc.cluster.local:27017' \
  --property rabbitmq.cluster='rabbitmq-0.rabbitmq-headless.default.svc.cluster.local:5672' \
  --property rabbitmq.user='guest' \
  --property rabbitmq.pass='guest' \
    infrastructure/inbound/src/main/java/it/bz/opendatahub/writer/WriterRoute.java
```

### Run the **Sample Pull Route (Südtirol Wine)**

```
kamel run \
  --name pull-route \
  --property rabbitmq.cluster='rabbitmq-0.rabbitmq-headless.default.svc.cluster.local:5672' \
  --property rabbitmq.user='guest' \
  --property rabbitmq.pass='guest' \
  --property pull.provider='suedtirol/wein2?fastline=false' \
  --property pull.endpoints='https://suedtirolwein.secure.consisto.net/companies.ashx,https://suedtirolwein.secure.consisto.net/awards.ashx' \
  --property pull.endpointKeys='companies,awards' \
    infrastructure/inbound/src/main/java/it/bz/opendatahub/pull/PullRoute.java
```

### Run the **Fastline Route (WebSocket)**

```
kamel run \
  --name fastline-route \
  --property rabbitmq.cluster='rabbitmq-0.rabbitmq-headless.default.svc.cluster.local:5672' \
  --property rabbitmq.user='guest' \
  --property rabbitmq.pass='guest' \
    infrastructure/router/src/main/java/it/bz/opendatahub/outbound/fastline/FastlineRoute.java
```

### Run the **Router Route (RabbitMQ)**

```
kamel run \
  --name router-route \
  --property rabbitmq.cluster='rabbitmq-0.rabbitmq-headless.default.svc.cluster.local:5672' \
  --property rabbitmq.user='guest' \
  --property rabbitmq.pass='guest' \
    infrastructure/router/src/main/java/it/bz/opendatahub/outbound/router/RouterRoute.java
```

### Run the **Update Route**

```
kamel run \
  --name update-route \
  --property rabbitmq.cluster='rabbitmq-0.rabbitmq-headless.default.svc.cluster.local:5672' \
  --property rabbitmq.user='guest' \
  --property rabbitmq.pass='guest' \
    infrastructure/router/src/main/java/it/bz/opendatahub/outbound/update/UpdateRoute.java
```

## Delete Camel Routes

It's not possible to scale down the Kamel integration from Kubernetes.
If we need to stop a route we have to tell Kamel to delete the whole route, which reflects in deleting the Kubernetes deployment.

```
kamel delete <route-name>
```

## Useful Resources

- [Camel Quarkus Extensions](https://camel.apache.org/camel-quarkus/2.16.x/reference/index.html)
- [Camel K Traits](https://camel.apache.org/camel-k/1.12.x/traits/traits.html)
