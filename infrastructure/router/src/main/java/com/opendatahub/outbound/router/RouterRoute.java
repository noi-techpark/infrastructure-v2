// SPDX-FileCopyrightText: NOI Techpark <digital@noi.bz.it>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// camel-k: dependency=camel:bean
// camel-k: dependency=camel:openapi-java
// camel-k: dependency=camel:paho
// camel-k: dependency=camel:spring-rabbitmq
// camel-k: dependency=camel:seda
// camel-k: dependency=camel:stream

package com.opendatahub.outbound.router;

import org.apache.camel.builder.RouteBuilder;
import org.apache.camel.component.springrabbit.SpringRabbitMQConstants;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.amqp.core.AmqpAdmin;
import org.springframework.amqp.core.Binding;
import org.springframework.amqp.core.Queue;
import org.springframework.amqp.core.TopicExchange;
import org.springframework.amqp.core.Binding.DestinationType;
import org.springframework.amqp.rabbit.connection.CachingConnectionFactory;
import org.springframework.amqp.rabbit.connection.ConnectionFactory;
import org.springframework.amqp.rabbit.core.RabbitAdmin;

import jakarta.enterprise.context.ApplicationScoped;
import java.util.Optional;

import org.eclipse.microprofile.config.ConfigProvider;

import org.apache.camel.model.dataformat.JsonLibrary;


/**
 * RabbitMQ configuration as defined by Quarkus.
 * <p>
 * The data in this interface is taken from System properties, ENV variables,
 * .env file and more. Take a look at https://quarkus.io/guides/config-reference
 * to see how it works.
 */
class RabbitMQConfig {
    String cluster;
    Optional<String> user;
    Optional<String> pass;
    String clientName;
}

class Payload {
    public String id;
    public String db;
    public String collection;
}

/**
 * Route to read from RabbitMQ.
 */
@ApplicationScoped
public class RouterRoute extends RouteBuilder {
    static final String RABBITMQ_READY_QUEUE = "ready-q";
    static final String RABBITMQ_READY_EXCHANGE = "ready";
    static final String RABBITMQ_ROUTED_QUEUE = "routed-q";
    static final String RABBITMQ_ROUTED_EXCHANGE = "routed";
    static final String RABBITMQ_UNROUTABLE_QUEUE = "routed-dl-q";
    static final String RABBITMQ_UNROUTABLE_EXCHANGE = "routed-dl";
    static final String RABBITMQ_CONNECTION_FACTORY = "default";

    private RabbitMQConfig rabbitConfig;
    private static Logger LOG = LoggerFactory.getLogger(RouterRoute.class);

    public RouterRoute()
    {
        this.rabbitConfig = new RabbitMQConfig();
        this.rabbitConfig.cluster = ConfigProvider.getConfig().getValue("rabbitmq.cluster", String.class);
        this.rabbitConfig.user = ConfigProvider.getConfig().getOptionalValue("rabbitmq.user", String.class);
        this.rabbitConfig.pass = ConfigProvider.getConfig().getOptionalValue("rabbitmq.pass", String.class);
        this.rabbitConfig.clientName = ConfigProvider.getConfig().getValue("rabbitmq.clientName", String.class);
    }

    public ConnectionFactory connectionFactory() {
        String user = this.rabbitConfig.user.orElseGet(() -> "*** no user ***");
        String pass = this.rabbitConfig.pass.map(p -> "*****").orElseGet(() -> "*** no password ***");

        LOG.info("RabbitMQ cluster: {}", this.rabbitConfig.cluster);
        LOG.info("RabbitMQ user: {}", user);
        LOG.info("RabbitMQ password: {}", pass);

        final CachingConnectionFactory fac = new CachingConnectionFactory();
        fac.setConnectionNameStrategy(_f -> rabbitConfig.clientName + ": " + System.getenv("HOSTNAME"));
        fac.setAddresses(rabbitConfig.cluster);
        if(user != null) {
            fac.setUsername(rabbitConfig.user.get());
            fac.setPassword(rabbitConfig.pass.get());
        }
        return fac;
    }

    @Override
    public void configure() {
        ConnectionFactory fac = connectionFactory();
        getCamelContext().getRegistry().bind(RABBITMQ_CONNECTION_FACTORY, fac);

        AmqpAdmin admin = new RabbitAdmin(fac);
        admin.declareExchange(new TopicExchange(RABBITMQ_ROUTED_EXCHANGE, true, false));
        admin.declareQueue(new Queue(RABBITMQ_ROUTED_QUEUE, true, false, false));
        admin.declareBinding(new Binding(RABBITMQ_ROUTED_QUEUE,DestinationType.QUEUE, RABBITMQ_ROUTED_EXCHANGE, "#", null));

        String RabbitMQConnectionString = getRabbitMQConnectionString();

        // Use RabbitMQ connection
        from(RabbitMQConnectionString)
                .routeId("[Route: RabbitMQ subscription]")
                // .log("RabbitMQ| ${body}")
                // .log("RabbitMQ| ${headers}")
                .unmarshal().json(JsonLibrary.Jackson, Payload.class)
                // .log("RabbitMQ| ${body}")
                .process(exchange -> {
                    Payload payload = (Payload) exchange.getMessage().getBody();
                    String routeKey = String.format("%s.%s", payload.db, payload.collection);
                    exchange.getMessage().setHeader(SpringRabbitMQConstants.ROUTING_OVERRIDE_KEY, routeKey);
                    exchange.getMessage().setHeader(SpringRabbitMQConstants.DEAD_LETTER_ROUTING_KEY, routeKey);
                })
                .marshal().json()
                .to(getRabbitMQRoutedConnectionString());
    }

    private String getRabbitMQConnectionString() {
        final StringBuilder uri = new StringBuilder(String.format("spring-rabbitmq:%s"+
            "?connectionFactory=#bean:%s" +
            "&queues=%s"+
            "&autoDeclare=true"+
            "&acknowledgeMode=AUTO"+
            "&exchangePattern=InOnly"+
            "&arg.queue.durable=true",
            RABBITMQ_READY_EXCHANGE, RABBITMQ_CONNECTION_FACTORY, RABBITMQ_READY_QUEUE));
        return uri.toString();
    }

    //TODO: autoDeclare doesn't work on consumers, declare this as a separate bean
    private String getRabbitMQRoutedConnectionString() {
        final StringBuilder uri = new StringBuilder(String.format("spring-rabbitmq:%s"+
            "?connectionFactory=#bean:%s" +
            "&queues=%s"+
            "&autoDeclare=true"+
            "&arg.queue.durable=true"+
            "&exchangeType=topic"+
            "&acknowledgeMode=AUTO"+
            "&exchangePattern=InOnly"+
            "&deadLetterExchange=%s"+
            "&deadLetterQueue=%s"+
            "&deadLetterExchangeType=fanout",
            RABBITMQ_ROUTED_EXCHANGE,
            RABBITMQ_CONNECTION_FACTORY,
            RABBITMQ_ROUTED_QUEUE,
            RABBITMQ_UNROUTABLE_EXCHANGE,
            RABBITMQ_UNROUTABLE_QUEUE));
        return uri.toString();
    }
}

@ApplicationScoped
class ErrorHandler extends RouteBuilder {

    private Logger LOG = LoggerFactory.getLogger(ErrorHandler.class);

    @Override
    public void configure() throws Exception {
        onCompletion()
                .onFailureOnly()
                .process(exchange -> LOG.error("{}", exchange.getMessage().getBody()));
    }
}
