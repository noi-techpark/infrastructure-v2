# Open-Data Hub Infrastructure

> This repository contains the new infrastructure architecture as described in the document prepared by Animeshon and approved by the owners of the project.

## Prerequisites

- [AWS CLI](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html)
- [Kuberentes CLI](https://kubernetes.io/docs/tasks/tools/)
- [Helm CLI](https://helm.sh/docs/intro/install/)
- [Terraform CLI](https://learn.hashicorp.com/tutorials/terraform/install-cli)

## Concepts

- [Terraform](docs/terraform.md)
- [Kubernetes](docs/kubernetes.md)
- [Helm](docs/helm.md)
- [Camel](docs/camel.md)

--- 

## Overview

This repository contains the PoC for Open Data Hub's new architecture.

 ![Architecture Overview](./docs/assets/Full%20Architecture.svg)

The PoC is designed and developed to run in two different environments:

- Locally for development purposes
  - In this configuration, the architecture is orchestrated with `docker-compose`
  - All `camel routes` are powered by **Java Quarkus** applications
  - The `notifier` is powered by a **Node.js** application
  - `RabbitMQ` is powered by the docker container **rabbitmq:3-management** which provides the Admin Pannel out of the box
- In the cloud as Staging environment
  - In this configuration, the architecture is orchestrated with `kubernetes`
  - All `camel routes` are powered by **[Camel K](docs/camel.md#camel-k)** applications
  - The `notifier` is a **dockerized Node.js** application
  - `RabbitMQ` is deployed using **RabbitMQ Cluster Operator for Kubernetes**

## Repository structure

<normal><pre>
├── infrastructure
│   ├── inbound
│   │   └── src/main/java/opendatahub
│   │       ├── inbound
│   │       │   ├── [mqtt/MqttRoute.java](./docs/componenets/mqtt-route)
│   │       │   └── [rest/RestRoute.java](./docs/componenets/rest-route)
│   │       ├── [pull/PullRoute.java](./docs/componenets/pull-route)
│   │       ├── [writer/WriterRoute.java](./docs/componenets/writer-route)
│   │       ├── [RabbitMQConnection.java](./docs/componenets/rabbitmq-connection)
│   │       └── [WrapperProcessor.java](./docs/componenets/wrapper-processor)
│   ├── notifier
│   │   └── src
│   │       ├── [changeStream.js](./docs/componenets/notifier#change-stream)
│   │       └── [main.js](./docs/componenets/notifier#main)
│   ├── router
│   │   └── src/main/java/opendatahub/outbound
│   │       ├── [fastline/FastlineRoute.java](./docs/componenets/fastline-route)
│   │       └── [router/RouterRoute.java](./docs/componenets/router-route)
│   ├── transformer
│   │   └── src/main/java/opendatahub/transformer
│   │       ├── [ConsumerImpl.java](./docs/componenets/transformer#consumer)
│   │       ├── Main.java
│   │       └── [Poller.java](./docs/componenets/transformer#poller)
</pre></normal>

## Local Quickstart
We provide a `docker-compose` file to start the architecture locally.

To run the cluster just 
```sh
docker-compose up
```
in the main folder. It will build and spin up all components.

The first time we compose-up, we have to initialize MongoDB's replica set. To do so we have to run the following command outside the cluster.

On linux machines:
```sh
docker exec odh-infrastructure-v2-mongodb1-1 mongosh --eval "rs.initiate({
            _id : 'rs0',
            members: [
              { _id : 0, host : 'mongodb1:27017' },
            ]
          })"
```

Using Docker Desktop on Windows:
```sh
docker exec odh-infrastructure-v2_mongodb1_1 mongosh --eval "rs.initiate({
            _id : 'rs0',
            members: [
              { _id : 0, host : 'mongodb1:27017' },
            ]
          })"
```

### Entrypoints

| Service | Address |
| - | - |
| Perimetral Mosquitto | localhost:1883 |
| Perimetral Rest | localhost:8080 |
| RabbitMQ Pannel | localhost:15672 |
| RabbitMQ AMPQ 0-9-1 port | localhost:5672 |
| MongoDB | localhost:27017 |

# How to make Requests and check the data flow
To make and listen to the MQTT brokers (perimetral or internal) we suggest using [MQTTX](https://mqttx.app/).
To make REST requests we suggest using [Insomnia](https://insomnia.rest/) or any other REST client.
To connect to the MongoDB deployment we suggest using [Compass](https://www.mongodb.com/products/compass). Be aware that being the deployment a `Replica Set`, the URI string must be properly configured ([Doc](https://www.mongodb.com/docs/manual/reference/connection-string/)) and you have to check **Direct Connection** in the **Advanced Connection Options** of Compass.

## What to do
Once all connections are established, you can subscribe to the `MQTT Brokers` and watch for messages or send, send `REST request` and connect to the `MongoDB` instance.

! All messages sent to the `Perimeter` **must be** valid JSON, otherwise, the `Integrator` will discard and log the message.

! All messages sent with MQTT **must be** sent with `QoS2`.


# Performances
Being a PoC, the whole system is meant to give an overall insight and overview of the proposed architecture.
Performances can be greatly improved using replicas, sharding, parallel programming, configuration tuning, and polish.

# Notifier
The `notifier`, written in JS and running on Node, is a good example of a possible bottleneck that can be greatly improved.
Instead of having a single instance subscripted to the whole `MongoDB deployment`, it could be split between multiple instances each one subscripted to a particular `MongoDB Database` or even to single `Collections`.
This kind of polish can be done only once the team decides how to distribute the `RawData` coming from different **Datasources**.

# Very important notes

## MQTT

We have to ensure that when an application disconnects, for any reason, from an MQTT / AmazonSNS broker, it gets all messages it missed while offline on reconnection.

Using Mosquitto isn't done by properly publishing messages and establishing a **Persistent Connection** to the Broker in the client side:

- Publishers MUST publish with `QoS` >= 1
- Subscribers MUST connect with `QoS` >= 1
- Subscribers MUST connect with `cleanStart` = false
- ALL Subscribers in ALL pods must connect with a unique `clientId` which can't change at pod restart


### Autoscaling
To create a persistent session with Mosquitto, each replica of the Client (might it be the `inbound MQTT` or the `Writer`) **has** to generate a `ClientID`.
This configuration property is particularly important when autoscaling is enabled.

`MQTT Client instance 1` => ClientID = 1

When autoscaling is performed

`MQTT Client instance 1` => ClientID = 1

`MQTT Client instance 2` => ClientID = 2

If `MQTT Client instance 2` restarts, the **ClientID** must remain `2`

***If two clients have the same ClientID, one will crash since the broker won't accept the connection. 
if no ClientID is provided all messages received by the broker while no client is connected will be lost.***


### Mosquitto Persistence

To ensure sessions and messages are persisted even if the `Mosquitto` restarts or goes offline, we have to properly configure the nodes.

in the `.conf` file provided to the mosquitto instance

`persistence true` has to be present.

other fields are available on the [man page](https://mosquitto.org/man/mosquitto-conf-5.html).

In the cloud the mosquitto deployment has to be managed as `stateful set` to claim a volume where to write the database needed by mosquitto to create a persistent instance.

## MQTT Message Throttling

When the pipeline fails to process a message, we have to make a decision:

- Send it to a special Storage if failure is related to a critical error (malformed JSON, unrecognized payload....)
- Retried if the failure is the responsibility of the infrastructure itself (MQTT broker offline, broken logic, ...).

To retry the message we have to introduce some **throttling** logic to delay the next time the message is pulled from the queue.

## Change Stream
To use the Change Stream feature, the MongoDB deployment MUST be deployed as `ReplicaSet`

## Notifier connection
The notifier subscribes to the MongoDB deployment and starts listening for changes.
In the case that the MongoDB deployment restarts / goes offline, the *Notifier* **MUST** implement a mechanism to check that the connection is alive and start reconnecting until the MongoDB deployment returns online.

## Writer and RawDataTable configuration
For the purpose of the PoC, we use a single MongoDB deployment as `rawDataTable` and we store data in `{provider}` **db** / `{provider}` **collection**
Example: provider = `flightdata` -> data stored in `flightdata/flightadata`.

If we need to use multiple deployments or custom paths, you have to write your own logic to connect and build connection strings in the **Writer**.

References:
- https://camel.apache.org/camel-quarkus/2.10.x/reference/extensions/mongodb.html
- https://quarkus.io/guides/mongodb
