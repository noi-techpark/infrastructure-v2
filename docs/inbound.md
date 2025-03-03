<!--
SPDX-FileCopyrightText: NOI Techpark <digital@noi.bz.it>

SPDX-License-Identifier: CC0-1.0
-->

# Inbounds

In this page the specifications for the data pushed/pulled by the **Inbound Apis** will be explained.

The concept illustrated applies both to data pushed by Providers onto Open Data Hub and to data pulled by Open Data Hub from providers regardless the protocol used (MQTT, REST, ...).

## Message wrapping
Any data ingested thought the inbounds is incapsulated in a standardized message using [WrapperProcessor](components/wrapper-processor) and sent To RabbitMQ.

## Provider URI

When a provider pushes data to Open Data Hub's inbounds, or when Open Data Hub collects data from a provider, the system needs to identify where the data comes from.

We designed the system to work with `Provider URI`s which identify who sent/from who we collect the data.

The `Provider URI` is specified in inbound and it's propagated to the whole system.

`Provider URI` has the same format of a standard **Web URI** and and should be decided using this [guideline](docs/guidelines.md#datacollectors-provider-standard).

## Dead Letters

Malformed `Provider URIs` , message which do not pass the validation specified by Open Data Hub's policies or *data collection failure* are routed to a **Dead Letter exchange** in [RabbitMQ](./rabbitmq.md) which collects all messaged ingested in the inbounds which are not compliant to Open Data Hub standards.
