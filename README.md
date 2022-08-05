# Open-Data Hub Infrastructure

> This repository contains the new infrastructure architecture as described in the document prepared by Animeshon and approved by the owners of the project.

## Prerequisites

- AWS CLI
- Helm CLI
- Terraform CLI
- Kubernetes CLI

## Links

- [Terraform Cloud](https://app.terraform.io/app/noi-digital/workspaces/opendatahub-v2)
- [Architecture]()
- [Kubernetes Dashboard](http://k8s-default-kubernet-62841e8dd0-301015478.eu-west-1.elb.amazonaws.com/)
- [Amazon EKS](https://eu-west-1.console.aws.amazon.com/eks/home?region=eu-west-1#/clusters/aws-main-eu-01)

## Terraform

- EKS
- Amazon SNS
- Amazon IoT

## Kubernetes & Helm

- Dashboard
- DataDog
- Apache Camel K
- MongoDB
- Change Stream Service

## Monitoring & Tracing

- [Kubernetes]()
- [MongoDB]()
- [Apache Camel K]()
- [Amazon SNS]()

## Guides

- [Terraform provisioning from scratch]()
- [Setup kubectl via AWS CLI]()
- [Add new user to Kubernetes]()
- [Expose services via ELB or ALB]()

## How to

### Terraform

```
cd infrastructure/terraform

terraform init
terraform plan
```

### Kubernetes Dashboard

https://artifacthub.io/packages/helm/metrics-server/metrics-server

```
helm repo add metrics-server https://kubernetes-sigs.github.io/metrics-server/
```

```
helm upgrade --install \
  metrics-server metrics-server/metrics-server \
  --values infrastructure/helm/metrics-server/values.yaml
```

https://artifacthub.io/packages/helm/k8s-dashboard/kubernetes-dashboard

```
helm repo add kubernetes-dashboard https://kubernetes.github.io/dashboard/
```

```
helm upgrade --install \
  kubernetes-dashboard kubernetes-dashboard/kubernetes-dashboard \
  --values infrastructure/helm/kubernetes-dashboard/values.yaml
```

### Amazon ALB in EKS

https://docs.aws.amazon.com/eks/latest/userguide/aws-load-balancer-controller.html

```
helm repo add eks https://aws.github.io/eks-charts
```

```
helm upgrade --install aws-load-balancer-controller eks/aws-load-balancer-controller \
  --values infrastructure/helm/kubernetes-dashboard/values.yaml \
  --namespace kube-system
```

### MongoDB

```
helm repo add bitnami https://charts.bitnami.com/bitnami
```

```
helm upgrade --install mongodb bitnami/mongodb \
  --values infrastructure/helm/mongodb/values.yaml
```

### Camel K

TODO: move docker repository to ECR once we have permissions.

```
kubectl create secret docker-registry docker-hub-secrets \
  --docker-username=[USER] \
  --docker-password=[TOKEN]
```

```
helm repo add camel-k https://apache.github.io/camel-k/charts
```

```
helm upgrade --install camel-k camel-k/camel-k \
  --values infrastructure/helm/camel-k/values.yaml
```

Tips: install `kamel` CLI and read Camel K [docs](https://camel.apache.org/camel-k/1.9.x/running/running.html).

#### Run Camel Routes

```
cd infrastructure
```

```
kamel run \
  --name mqtt-route \
  --property mqtt.url=tcp://mosquitto-edge:1883 \
  --property internal_mqtt.url=tcp://mosquitto-storage.default.svc.cluster.local:1883 \
  --property internal_mqtt.topic=storage \
    inbound/src/main/java/it/bz/opendatahub/inbound/MqttRoute.java
```

```
kamel run \
  --name rest-route \
  --property internal_mqtt.url=tcp://mosquitto-storage.default.svc.cluster.local:1883 \
  --property internal_mqtt.topic=storage \
    inbound/src/main/java/it/bz/opendatahub/inbound/RestRoute.java
```

```
kamel run \
  --name writer-route \
  --property quarkus.mongodb.connection-string=mongodb://mongodb-0.mongodb-headless.default.svc.cluster.local:27017 \
  --property quarkus.mongodb.devservices.enabled=false \
  --property internal_mqtt.url=tcp://mosquitto-storage.default.svc.cluster.local:1883 \
  --property internal_mqtt.topic=storage \
    writer/src/main/java/it/bz/opendatahub/writer/WriterRoute.java
```

### Mosquitto

```
helm repo add k8s-at-home https://k8s-at-home.com/charts/
```

```
helm upgrade --install mosquitto-edge k8s-at-home/mosquitto \
  --values infrastructure/helm/mosquitto/values-edge.yaml

helm upgrade --install mosquitto-notifier k8s-at-home/mosquitto \
  --values infrastructure/helm/mosquitto/values-notifier.yaml

helm upgrade --install mosquitto-storage k8s-at-home/mosquitto \
  --values infrastructure/helm/mosquitto/values-storage.yaml
```

### Notifier

TODO: move docker repository to ECR once we have permissions.

```
docker build -t roggia/notifier:latest .
docker push roggia/notifier:latest
```

```
helm upgrade --install notifier ./infrastructure/helm/notifier/notifier \
  --values infrastructure/helm/notifier/values.yaml
```

--- 

## Overview


The prokect is build using Quarkus and Maven for all Camel routes and Node for the subscriber.

### Camel K

When deplyed in Kubernetes the Camel routes are deplyed using Camel K.
Camel K do not deploy the wole project but ony the route definition. To do so it needs a special syntax in the .java files to know which package to install.

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

It's also recommend to keep the route in a single file.
Reference: [Blog Article](https://piotrminkowski.com/2020/12/08/apache-camel-k-and-quarkus-on-kubernetes/).

### Docker
We provide a `docker-compose` file to start the architecture locally.

To run the custer just 
```sh
docker-compose up
```
in the main folder. It will build and spin up all components.

The first time we compose up, we have to initialize MongoDB's replica set. To do so we have to run the command outside the cluster.

```sh
docker exec odh-infrastructure-v2_mongodb1_1 mongo --eval "rs.initiate({
            _id : 'rs0',
            members: [
              { _id : 0, host : 'mongodb1:27017' },
            ]
          })"
```

`perimetral Mosquitto` exposet at: `localhost:1883`

`perimetral Rest` exposet at: `localhost:8080`

`internal Mosquitto` exposet at: `localhost:1884`

`transformers Mosquitto` exposet at: `localhost:1885`

`MongoDB` exposet at: `localhost:27017`

*Note that when using docker compose, we are not deploying using **Camel K** but building a docker container in which a full Maven-Quarkus application runs.*

*Docker compose deplyoment uses **Mosquitto** instad of **AmazonSNS***

# Very important notes

## MQTT

We have to ensure that when an application disconnects, for any reason, from an MQTT / AmazonSNS broker, it gets all messages it missed while offline on reconnection.

Using Mosquitto is't done by properly publishing mesage and establishing a **Persistent Connection** to the Broker in the client side:

- Publishers MUST publish with `QoS` >= 1
- Subscribers MUST connect with `QoS` >= 1
- Subscribers MUST connect with `cleanStart` = false
- ALL Subscribers in ALL pods must connect with an unique `clientId` which can't change at pod restart


## MQTT Message ACK

All service/routes relying on a MQTT queue as datasource MUST ensure the message is *Acknowledged* (**ACK**) only when it finishes to process the message.
Any message Acknowledged before the end of the flow might go lost if the application restarts or Exception/Error occurs during the process.

## MQTT Message Throtling

Wen the pipeline fails to process a message, we have to make a decision:

- Send it to a special Storage if failure is related to a critical error (malformed JSON, unrecognized payload....)
- Retried if the failure is responsability of the infrastructure itself (MQTT broker offline, broken logic, ...).

To retry the message we have to introduce some **throtling** logic to delay the next time the message is pulled from the queue.

## Change Stream
To use the Change Stream feature, the MongoDB deplyment MUST be deployed as `ReplicaSet`

## Notifier connection
The notifier subscribes to the MongoDB deplyoment ans start listening for changes.
Before it might happen the deplyment istelf restarts / goes offline, the *Notiier* **MUST** provide a mechanism to check the connection and reconnection to the Deplyoment.

## Writer and RawDataTable configuration
For the purpose of the PoC, we use a single MongoDB deployment as `rawDataTable` and we store data in `{provider}` **db** / `{provider}` **collection**
Example: provider = `flightdata` -> data stored in `flightdata/flightadata`.

If we need to use multiple deplyoments or custom paths, you have to write your own logic to connect and build connection strings in the **Writer**.

References:
- https://camel.apache.org/camel-quarkus/2.10.x/reference/extensions/mongodb.html
- https://quarkus.io/guides/mongodb
