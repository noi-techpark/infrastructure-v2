// SPDX-FileCopyrightText: NOI Techpark <digital@noi.bz.it>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-bean
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-openapi-java
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-paho
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-rabbitmq
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-seda
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-stream

package com.opendatahub.outbound.router;

import org.apache.camel.builder.RouteBuilder;
import org.apache.camel.component.springrabbit.SpringRabbitMQConstants;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import jakarta.enterprise.context.ApplicationScoped;

import java.time.ZoneId;
import java.time.ZonedDateTime;
import java.time.format.DateTimeFormatter;
import java.util.HashMap;
import java.util.Map;

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

    // Username is optional and may not be set
    Optional<String> user;

    // Password is optional and may not be set
    Optional<String> pass;
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

    private RabbitMQConfig RabbitMQConfig;

    public RouterRoute()
    {
        this.RabbitMQConfig = new RabbitMQConfig();
        this.RabbitMQConfig.cluster = ConfigProvider.getConfig().getValue("rabbitmq.cluster", String.class);
        this.RabbitMQConfig.user = ConfigProvider.getConfig().getOptionalValue("rabbitmq.user", String.class);
        this.RabbitMQConfig.pass = ConfigProvider.getConfig().getOptionalValue("rabbitmq.pass", String.class);
    }

    @Override
    public void configure() {
        RabbitMQConfigLogger.log(RabbitMQConfig);

        String RabbitMQConnectionString = getRabbitMQConnectionString();

        // Use RabbitMQ connection
        from(RabbitMQConnectionString)
                .routeId("[Route: RabbitMQ subscription]")
                // .log("RabbitMQ| ${body}")
                // .log("RabbitMQ| ${headers}")
                .unmarshal().json(JsonLibrary.Jackson, Payload.class)
                // .log("RabbitMQ| ${body}")
                .process(exchange -> {
                    Payload payload = (Payload)exchange.getMessage().getBody();
                    String routeKey = String.format("%s.%s", payload.db, payload.collection);
                    exchange.getMessage().setHeader(SpringRabbitMQConstants.ROUTING_KEY, routeKey);
                    exchange.getMessage().setHeader(SpringRabbitMQConstants.RABBITMQ_DEAD_LETTER_ROUTING_KEY, routeKey);
                })
                .marshal().json()
                .to(getRabbitMQRoutedConnectionString());
    }

    private String getRabbitMQConnectionString() {
        final StringBuilder uri = new StringBuilder(String.format("spring-rabbitmq:%s?"+
            "addresses=%s"+
            "&queue=%s"+
            "&autoAck=false"+
            "&autoDelete=false",
            RABBITMQ_READY_EXCHANGE, RabbitMQConfig.cluster, RABBITMQ_READY_QUEUE));

        // Check if RabbitMQ credentials are provided. If so, then add the credentials to the connection string
        RabbitMQConfig.user.ifPresent(user -> uri.append(String.format("&username=%s", user)));
        RabbitMQConfig.pass.ifPresent(pass -> uri.append(String.format("&password=%s", pass)));

        System.out.println(uri.toString());

        return uri.toString();
    }

    private String getRabbitMQRoutedConnectionString() {
        final StringBuilder uri = new StringBuilder(String.format("spring-rabbitmq:%s?"+
            "addresses=%s"+
            "&queue=%s"+
            "&autoDelete=false"+
            "&exchangeType=topic"+
            "&deadLetterExchange=%s"+
            "&deadLetterQueue=%s"+
            "&deadLetterExchangeType=fanout"+
            "&arg.exchange.alternate-exchange=%s",
            RABBITMQ_ROUTED_EXCHANGE,
            RabbitMQConfig.cluster,
            RABBITMQ_ROUTED_QUEUE,
            RABBITMQ_UNROUTABLE_EXCHANGE,
            RABBITMQ_UNROUTABLE_QUEUE,
            RABBITMQ_UNROUTABLE_EXCHANGE
            ));

        // Check if RabbitMQ credentials are provided. If so, then add the credentials to the connection string
        RabbitMQConfig.user.ifPresent(user -> uri.append(String.format("&username=%s", user)));
        RabbitMQConfig.pass.ifPresent(pass -> uri.append(String.format("&password=%s", pass)));

        System.out.println(uri.toString());
        return uri.toString();
    }
}

final class RabbitMQConfigLogger {

    private static Logger LOG = LoggerFactory.getLogger(RabbitMQConfigLogger.class);

    private RabbitMQConfigLogger() {
        // Private constructor, don't allow new instances
    }

    public static void log(RabbitMQConfig config) {
        String user = config.user.orElseGet(() -> "*** no user ***");
        String pass = config.pass.map(p -> "*****").orElseGet(() -> "*** no password ***");

        LOG.info("RabbitMQ cluster: {}", config.cluster);
        LOG.info("RabbitMQ user: {}", user);
        LOG.info("RabbitMQ password: {}", pass);
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
