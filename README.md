<!--
SPDX-FileCopyrightText: NOI Techpark <digital@noi.bz.it>

SPDX-License-Identifier: CC0-1.0
-->

# Open Data Hub Infrastructure

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
- [RabbitMQ](docs/rabbitmq.md)
- [Provider URI](docs/inbound.md#provider-uri)

## Tools

* [Remote RabbitMQ Queue Manager](docs/devops/rabbitmq_dev_route.md)

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
│   │       ├── [writer/WriterRoute.java](./docs/components/writer-route.md)
│   ├── notifier
│   │   └── src
│   │       ├── [changeStream.js](./docs/components/notifier.md#change-stream)
│   │       └── [main.js](./docs/components/notifier.md#main)
│   ├── router
│   │   └── src/main/java/opendatahub/outbound
│   │       ├── [fastline/FastlineRoute.java](./docs/components/fastline-route.md)
│   │       ├── [update/UpdateRoute.java](./docs/components/push-update-route.md)
│   │       └── [router/RouterRoute.java](./docs/components/router-route.md)
│   ├── transformer
│   │   └── src/main/java/opendatahub/transformer
│   │       ├── [ConsumerImpl.java](./docs/components/transformer.md#consumer)
│   │       ├── Main.java
│   │       └── [Poller.java](./docs/components/transformer.md#poller)
</pre></normal>

## Local Quickstart
We provide a `docker-compose` file to start the architecture locally.

To run the cluster just 
```sh
docker compose up
```
in the main folder. It will build and spin up all components.

### Entrypoints

| Service | Address |
| - | - |
| RabbitMQ Panel | localhost:15672 |
| RabbitMQ AMPQ 0-9-1 port | localhost:5672 |
| MongoDB | localhost:27017 |

# How to make Requests and check the data flow
To connect to the MongoDB deployment we suggest using [Compass](https://www.mongodb.com/products/compass). Be aware that being the deployment a `Replica Set`, the URI string must be properly configured ([Doc](https://www.mongodb.com/docs/manual/reference/connection-string/)) and you have to check **Direct Connection** in the **Advanced Connection Options** of Compass.

## Change Stream
To use the Change Stream feature, the MongoDB deployment MUST be deployed as `ReplicaSet`

# Troubleshooting

The following section is dedicated to the troubleshooting of known or common issues.

## Docker Compose

There are a number of common issues related to docker-compose that might prevent the correct setup of the local environment. Verify the following configurations to be appropriate depending on your local runtime environment and operating system:

- The `docker` and `docker-compose` versions are up-to-date
- The `docker` caches are not interfering with a newer configuration (run `docker system prune --all` to delete all volumes, images, and containers)
- The branch is up-to-date (run `git pull`)
- The permissions assigned to the mounted volumes (folders) on your host are valid (MongoDB is especially known to generate issues with the permissions)
- Windows line endings are not interfering with bash scripts and tools configurations (run `git rm --cached -r . && git reset --hard` to force line endings normalization) - this issue usually affects only builds on Windows systems, please read visit the official Git documentation to learn more about [gitattributes](https://git-scm.com/docs/gitattributes) and line endings normalization.
