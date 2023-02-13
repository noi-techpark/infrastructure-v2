# RabbitMQ Connection

**RabbitConnection.java** is just an utils class to configure `Camel RabbitMQ components`.

The configuration is made by a single string and allow Routes to communicate with Open Data Hub's core RabbitMQ cluster.

For the comprehensive documentation about the configuration, see [Apache Camel's RabbitMQ component documentation](https://camel.apache.org/components/3.20.x/spring-rabbitmq-component.html).

### Resources

[ExchangePattern](https://stackoverflow.com/questions/14527185/activemq-i-cant-consume-a-message-sent-from-camel-using-inout-pattern)

## Example 
Let's make a real world example:

```java
public String getRabbitMQIngressConnectionString() {
        final StringBuilder uri = new StringBuilder(String.format("rabbitmq:%s?"+
            "addresses=%s"+
            "&queue=%s"+
            "&autoAck=false"+
            // setting reQueue=true + autoAck=false messages not processed because of exceptions get requeued
            "&reQueue=true"+ 
            "&routingKey=ingress.#"+
            // "&deadLetterExchange=%s"+
            "&passive=false"+
            // "&skipExchangeDeclare=true"+
            // "&skipQueueBind=true"+
            "&autoDelete=false"+
            "&publisherAcknowledgements=false"+
            "&exchangeType=topic"+
            "&exchangePattern=InOnly"+
            "&declare=true",
            this.ingressConfig.ingressTopic,
            this.ingressConfig.cluster,
            this.ingressConfig.ingressQueue/*,
            this.ingressConfig.ingressDLTopic*/));

        // Check if RabbitMQ credentials are provided. If so, then add the credentials to the connection string
        this.ingressConfig.user.ifPresent(user -> uri.append(String.format("&username=%s", user)));
        this.ingressConfig.pass.ifPresent(pass -> uri.append(String.format("&password=%s", pass)));

        System.out.println(uri.toString());
        return uri.toString();
    }
```
The above connection string is used by both **publishers**([MQTT Route](mqtt-route.md), [REST Route](rest-route.md) or [Pull Route](pull-route.md)) and **consumers** ([Writer Route](writer-mqtt-route.md)).

```java
"rabbitmq:%s?" // PUBLISHERS - the exchange where to publish
```

```java
"addresses=%s" // PUBLISHERS & CONSUMERS - addresses of the rabbit nodes
``096

```java
"&queue=%s" // CONSUMERS - The queue to which subscribe
```

```java
"&reQueue=true" // CONSUMERS - If true the route re enqueue messages when the route fails to complete
```

```java
"&routingKey=ingress.#" // CONSUMERS - The route key which binds the exchange with the queue
```

```java
"&autoDelete=false" // PUBLISHERS & CONSUMERS - Do not delete queue & exchange when the application exists, creating therefore persistent queues and exchanges
```

...