// SPDX-FileCopyrightText: NOI Techpark <digital@noi.bz.it>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// camel-k: dependency=camel:bean
// camel-k: dependency=camel:openapi-java
// camel-k: dependency=camel:paho
// camel-k: dependency=camel:spring-rabbitmq
// camel-k: dependency=camel:seda
// camel-k: dependency=camel:stream
// camel-k: dependency=camel:vertx-websocket

// camel-k: trait=ingress.enabled=false
// camel-k: trait=service.enabled=true trait=service.type=NodePort

package com.opendatahub.outbound.fastline;

import org.apache.camel.builder.RouteBuilder;
import org.apache.camel.component.springrabbit.SpringRabbitMQConstants;

import jakarta.enterprise.context.ApplicationScoped;

import java.util.Optional;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.amqp.rabbit.connection.CachingConnectionFactory;
import org.springframework.amqp.rabbit.connection.ConnectionFactory;
import org.eclipse.microprofile.config.ConfigProvider;
import org.apache.camel.component.vertx.websocket.VertxWebsocketConstants;

class RabbitMQConfig {
    String cluster;
    Optional<String> user;
    Optional<String> pass;
    String clientName;
}

/**
 * Route to read from RabbitMQ.
 */
@ApplicationScoped
public class FastlineRoute extends RouteBuilder {
    static final String RABBITMQ_FASTLINE_QUEUE = "fastline-q";
    static final String RABBITMQ_FASTLINE_EXCHANGE = "fastline";
    static final String RABBITMQ_CONNECTION_FACTORY = "fastline";

    private RabbitMQConfig rabbitConfig;
    private Optional<Boolean> isLocal;
    private static Logger LOG = LoggerFactory.getLogger(FastlineRoute.class);

    public FastlineRoute()
    {
        this.rabbitConfig = new RabbitMQConfig();
        this.rabbitConfig.cluster = ConfigProvider.getConfig().getValue("rabbitmq.cluster", String.class);
        this.rabbitConfig.user = ConfigProvider.getConfig().getOptionalValue("rabbitmq.user", String.class);
        this.rabbitConfig.pass = ConfigProvider.getConfig().getOptionalValue("rabbitmq.pass", String.class);
        this.isLocal = ConfigProvider.getConfig().getOptionalValue("local", Boolean.class);
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
        getCamelContext().getRegistry().bind(RABBITMQ_CONNECTION_FACTORY, connectionFactory());

        // Use RabbitMQ connection
        from(getRabbitMQConnectionString())
            .routeId("[Route: fastline]")
            .process(exchange -> {
                String destination = "";
                String route = exchange.getMessage().getHeader(SpringRabbitMQConstants.ROUTING_KEY).
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
        final StringBuilder uri = new StringBuilder(String.format("spring-rabbitmq:%s"+
            "?connectionFactory=#bean:%s" +
            "&queues=%s"+
            "&autoDeclare=true"+
            "&acknowledgeMode=MANUAL"+
            "&routingKey=#"+ // any routing key
            "&exchangeType=topic"+
            "&exchangePattern=InOnly"+
            "&arg.queue.durable=true",
            RABBITMQ_FASTLINE_EXCHANGE, RABBITMQ_CONNECTION_FACTORY, RABBITMQ_FASTLINE_QUEUE));
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
