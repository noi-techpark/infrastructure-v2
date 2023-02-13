# Transformers

![transformer](../assets/transformer.svg)

**Transformers** are applications that transform raw data stored in the [Raw Data Table](../raw-data-table.md) and insert the result in [Data Marts](../data-marts.md).

Transformers implementation is completely domain dependent and it follows custom logic for each data type ingested in the system.

There are some behaviors all **transformers** must complain to:

## RabbitMQ Connector

Each **transformer** must connect to a [RabbitMQ Queue](../rabbitmq.md#queue) and wait for new messages.

To connect to RabbitMQ just use a client implementation in the transformer target language.

## MongoDB Connector

The messages polled from RabbitMQ **do not contain** the data, but only a document reference in the [Raw Data Table](../raw-data-table.md).

The **transformer** has therefore the responsibility to retrieve the document from the Raw Data Table using a MongoDB client implementation in the transformer target language.

## Poller

The **Transformer** must poll messages from RabbitMQ and [acknowledge](../rabbitmq.md#ack) or [NACK](../rabbitmq.md#nack) them depending on the transformation outcome.

In some cases, the transformer needs more messages to build a complete dataset (Flightdata needs 4 different partial data to build a complete state of a flying aircraft), in such cases the **poller** class of the **transformer** should store in-memory the data for a fixed amount of time, hoping to get all pieces of information necessary to perform a transformation.

If a configured `deadline` is met, the **transformer** should **NACK** all messages before RabbitMQ flags the transformer as `not responsive` and disconnects it.

## Consumer

The consumer is just a function, a class or a flow inside the **transformer** which takes one or more messages and produces a map of "processed messages". All successfully processed messages must be `acked` (to delete them from the system, since they have been successfully processed) and all messages which can be later retried must be `nacked`.

Messages which have **permanently failed the transformation** (for any reason) should be `acked` and logged. If we don't ack them, the system will keep trying to process them, clogging the transformer.

## Post Hook

In some cases, the transformer wants to inform the system that it has finished transforming some data which is now available in a **Data Mart**.

To do so all **transformers** must provide a mechanism to send back to RabbitMQ one or more messages for each successful message before its `ack`.

The **post hook message** can be the route to an outbound API, or it could be captured from another transformer to perform further computation.
