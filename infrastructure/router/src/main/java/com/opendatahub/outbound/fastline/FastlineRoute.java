// SPDX-FileCopyrightText: NOI Techpark <digital@noi.bz.it>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-bean
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-openapi-java
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-paho
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-rabbitmq
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-seda
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-stream
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-vertx-websocket

// camel-k: trait=ingress.enabled=false
// camel-k: trait=service.enabled=true trait=service.type=NodePort

package com.opendatahub.outbound.fastline;

import org.apache.camel.builder.RouteBuilder;
import org.apache.camel.component.rabbitmq.RabbitMQConstants;

import javax.enterprise.context.ApplicationScoped;

import java.util.Optional;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import org.eclipse.microprofile.config.ConfigProvider;
import org.apache.camel.component.vertx.websocket.VertxWebsocketConstants;

class RabbitMQConfig {
    String cluster;

    String ingressQueue;

    // Username is optional and may not be set
    Optional<String> user;

    // Password is optional and may not be set
    Optional<String> pass;
}

/**
 * Route to read from RabbitMQ.
 */
@ApplicationScoped
public class FastlineRoute extends RouteBuilder {
    static final String RABBITMQ_FASTLINE_QUEUE = "fastline-q";
    static final String RABBITMQ_FASTLINE_EXCHANGE = "fastline";

    private RabbitMQConfig RabbitMQConfig;
    private Optional<Boolean> isLocal;

    public FastlineRoute()
    {
        this.RabbitMQConfig = new RabbitMQConfig();
        this.RabbitMQConfig.cluster = ConfigProvider.getConfig().getValue("rabbitmq.cluster", String.class);
        this.RabbitMQConfig.user = ConfigProvider.getConfig().getOptionalValue("rabbitmq.user", String.class);
        this.RabbitMQConfig.pass = ConfigProvider.getConfig().getOptionalValue("rabbitmq.pass", String.class);
        this.isLocal = ConfigProvider.getConfig().getOptionalValue("local", Boolean.class);
    } 

    @Override
    public void configure() {
        String RabbitMQConnectionString = getRabbitMQConnectionString();

        // Use RabbitMQ connection
        from(RabbitMQConnectionString)
            .routeId("[Route: fastline]")
            .process(exchange -> {
                String destination = "";
                String route = exchange.getMessage().getHeader(RabbitMQConstants.ROUTING_KEY).
                    toString().
                    replaceAll("\\.", "/");
                if (this.isLocal.isPresent()) {
                    destination = String.format("websocket://0.0.0.0:8081/fastline?sendToAll=true,"+
                        "websocket://0.0.0.0:8081/fastline/%s?sendToAll=true",
                        route
                    );
                } else {
                    // Kamel
                    exchange.getMessage().setHeader(VertxWebsocketConstants.SEND_TO_ALL, true);
                    destination = String.format("vertx-websocket://0.0.0.0:8081/fastline,"+
                        "vertx-websocket://0.0.0.0:8081/fastline/%s",
                        route
                    );
                }
                
                System.out.println(destination);
                exchange.getMessage().setHeader("destination", destination);
            })
            .recipientList(header("destination"));
    }

    private String getRabbitMQConnectionString() {
        final StringBuilder uri = new StringBuilder(String.format("rabbitmq:%s?"+
            "addresses=%s"+
            "&queue=%s"+
            "&routingKey=#"+ // any routing key
            "&exchangeType=topic"+
            "&autoDelete=false", 
            RABBITMQ_FASTLINE_EXCHANGE, RabbitMQConfig.cluster, RABBITMQ_FASTLINE_QUEUE));

        // Check if RabbitMQ credentials are provided. If so, then add the credentials to the connection string
        RabbitMQConfig.user.ifPresent(user -> uri.append(String.format("&username=%s", user)));
        RabbitMQConfig.pass.ifPresent(pass -> uri.append(String.format("&password=%s", pass)));

        System.out.println(uri.toString());

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
