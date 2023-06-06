<!--
SPDX-FileCopyrightText: NOI Techpark <digital@noi.bz.it>

SPDX-License-Identifier: CC0-1.0
-->

# RabbitMQ

RabbitMQ is an event bus composed of mainly 3 concepts:

- [Exchanges](#exchange)
- [Queues](#queue)
- [Bindings](#bind)

The system uses `AMP 0-9-1` protocol to interact with RabbitMQ.

RabbitMQ provides an Administration Panel to have a comprehensive overview of the cluster status. The Panel is available on `15672` port of the cluster.
(while working locally it is available at the address http://localhost:15672/)

## Exchange

Exchanges are the entrypoints to RabbitMQ, when *publishing*, a producer is interacting with an `exchange`.

There are 3 types of `exchanges`:

- Direct exchanges, which route messages to a single `queue`/consumer/exchange based on the [Routing Key](#routing-key)
- Fanout exchanges, which broadcast messages to all subscribed  `queue`/consumer/exchange
- **Topic exchanges**, which route messages to multiple `queue`/consumer/exchange based on the [Routing Key](#routing-key) allowing route key regex filtering

Since the system utilizes mostly the **topic exchange**, we suggest reading more about them in the official [documentation](https://www.rabbitmq.com/tutorials/amqp-concepts.html) and [tutorials](https://www.rabbitmq.com/tutorials/tutorial-five-python.html).

When a message enters an `exchange` it is immediately broadcasted to all subscribed `queue`/consumer/exchange if any.

If there are no subscribers, the message is simply lost.

## Queue

Queues are buffers where messages can be stored waiting to be consumed.

In the most common configuration, an **exchange** is **bounded** to a `queue`.

In this way, messages published to the exchange are not thrown away on the fly; but persisted in a `queue` waiting to be processed.

### **Queue size limit**

By default queues do not have a limit (message number or bytes). For more information about Queue's length limit read the [official documentation](https://www.rabbitmq.com/maxlength.html).

## Message Acknowledgment

When a Route or [Transformer](./components/transformer.md) pulls a message, it becomes responsible to tell RabbitMQ how the message process went.

It has a timeout (configure [consumer_timout docs](https://www.rabbitmq.com/consumers.html#acknowledgement-timeout)) to give a response about the message. After the timeout RabbitMQ flags the consumer as **non-responsive**, disconnects it and sends the messages he was responsible for to other consumers.

There are 3 types of aknowledgement:

- `ack`\
    deletes the message from the system, since it has been successfully processed

- `reject`\
    deletes the message from the system syem because a permanent error occurred.\
    If a [dead letter exchange](https://www.rabbitmq.com/dlx.html#routing) is configured, all rejected messages will forwarded to that exchange

- `nack` (negative ack)\
    There was a retriable error in the process, therefore the message is requeued to further retry.

## Bind

Bindings are the glue between `exchanges` and `queues`.

More technically a `bind` creates a rule to tell an exchange when to forward a message to a specific queue. 

The most basic **bind** simply tells the queue to broadcast all messages to a queue. 

When declaring a **bind** it is possible to specify a [Routing Key](#routing-key) to tell the exchange to broadcast only those messages routed with that **Routing Key**.

**An exchange can have multiple bindings with a queue.**

To know more about `binding` we suggest reading the official [tutorials](https://www.rabbitmq.com/tutorials/tutorial-three-python.html).

## Routing Key

**Routing Keys** are a very important concept in RabbitMQ. As the name suggest, by configuring a `bind` to allow only messages flagged with a specific **HEADER** to be forwarded to the target **queue**.

When bounded with a `topic exchange`, it is also possible to use wildcards in the routing key, allowing more powerful routing logic.

![topic routing](https://www.rabbitmq.com/img/tutorials/python-five.png)

In the above example, messages published with a routing key will be routed so:

| Message Routing Key | Destination Queue |
| - | - | 
| an.orange.yay | Q1 |
| a.orange | unroutable | 
| a.nice.rabbit | Q2 |
| lazy | Q2 |
| lazy.guy.singing | Q2 |

```
Topic exchange is powerful and can behave like other exchanges.

When a queue is bound with "#" (hash) binding key - it will receive all the messages, regardless of the routing key - like in fanout exchange.

When special characters "*" (star) and "#" (hash) aren't used in bindings, the topic exchange will behave just like a direct one.
```

Read more info about **Routing Keys** in the [tutorials](https://www.rabbitmq.com/tutorials/tutorial-five-python.html)

References:
- https://www.rabbitmq.com/tutorials/amqp-concepts.html
- https://www.rabbitmq.com/tutorials/tutorial-one-python.html