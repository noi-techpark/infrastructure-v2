// SPDX-FileCopyrightText: NOI Techpark <digital@noi.bz.it>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// camel-k: dependency=mvn:io.quarkus:quarkus-mongodb-client
// camel-k: dependency=camel:jackson
// camel-k: dependency=camel:bean
// camel-k: dependency=camel:jackson
// camel-k: dependency=camel:mongodb
// camel-k: dependency=camel:paho
// camel-k: dependency=camel:spring-rabbitmq
// camel-k: dependency=mvn:org.apache.commons:commons-lang3:3.12.0


package com.opendatahub.writer;

import java.net.URI;
import java.net.URISyntaxException;
import java.time.Instant;
import java.util.Date;
import java.util.HashMap;
import java.util.Map;
import java.util.Optional;
import java.util.stream.Collectors;

import jakarta.enterprise.context.ApplicationScoped;

import org.apache.camel.builder.RouteBuilder;
import org.apache.camel.component.jackson.JacksonDataFormat;
import org.eclipse.microprofile.config.ConfigProvider;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.amqp.rabbit.connection.CachingConnectionFactory;
import org.springframework.amqp.rabbit.connection.ConnectionFactory;

/**
 * Route to read from INTERNAL MQTT and store data in rawDataTable.
 */
@ApplicationScoped
public class WriterRoute extends RouteBuilder {
    private final RabbitMQConnection rabbitMQConfig;
    private final Logger log = LoggerFactory.getLogger(WriterRoute.class);

    public WriterRoute()
    {
        this.rabbitMQConfig = new RabbitMQConnection();
    } 

    @Override
    public void configure() {
        getCamelContext().getRegistry().bind(RabbitMQConnection.CONNECTION_FACTORY, rabbitMQConfig.connectionFactory());
        // Read from RabbitMQ ingress
        // Writes a valid BSON object to MongoDB
        // TODO Add throtling if needed
        // https://camel.apache.org/components/3.18.x/rabbitmq-component.html
        from(this.rabbitMQConfig.getRabbitMQIngressFrom())
            .routeId("[Route: Writer]")
            //.throttle(100).timePeriodMillis(10000)
            // .log("WRITE| ${body}")
            .unmarshal(new JacksonDataFormat())
            .process(exchange -> {
                // First we unmarshal the payload
                Map<String, Object> body = (HashMap<String, Object>)exchange.getMessage().getBody(Map.class);
                Object timestamp = body.get("timestamp");
                // we convert the timestamp field into a valid BSON TimeStamp
                if (timestamp != null)
                {
                    Instant instant = Instant.parse((String)timestamp);
                    Date dateTimestamp = Date.from(instant);
                    body.put("bsontimestamp", dateTimestamp);
                }
                exchange.getMessage().setBody(body);
                // we then compute the database connection using the message body (in this case we only care bout the field `provider`)
                // and store the connection string in the `database` header to be used later
                exchange.getMessage().setHeader("database", getDatabaseString((String)body.get("provider")));
            })
            // we don't use `.to()` because the connection string is dynamic and we use the previously set header `database`
            // to send the data to the database
            .recipientList(header("database"))
            .end();
    }

    /**
     * For the purpose of the PoC, we use a single MongoDB deployment as rawDataTable.
     * The db name is the first token of the provider uri
     * The collection name is the second token of the provider uri
     *      if there is only one token it will be used as collection as well
     * 
     * If we need to use multiple deployments or custom paths, you should edit this function.
     * References:
     * https://camel.apache.org/components/3.20.x/mongodb-component.html
     * https://quarkus.io/guides/mongodb
     */
    // ! invalid collection & db characters (on linux deployment): /\. "$ 
    // https://www.mongodb.com/docs/manual/reference/limits/#std-label-restrictions-on-db-names
    private String getDatabaseString(String provider) throws URISyntaxException {
        String[] tokens = new URI(provider).getPath().split("/");
        String db = tokens[0];
        String collection = tokens[0];
        if (tokens.length > 1) {
            collection = tokens[1];
        }
        // connection client is created by quarkus
        final StringBuilder uri = new StringBuilder(String.format("mongodb:connectionBean?database=%s&collection=%s&operation=insert", db, collection));
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

class RabbitMQConfig {
    String cluster;
    Optional<String> user;
    Optional<String> pass;
    String clientName;
}

class RabbitMQConnection {
    static final String RABBITMQ_INGRESS_QUEUE = "ingress-q";
    static final String RABBITMQ_INGRESS_EXCHANGE = "ingress";
    static final String RABBITMQ_INGRESS_DEADLETTER_QUEUE = "ingress-dl-q";
    static final String RABBITMQ_INGRESS_DEADLETTER_EXCHANGE = "ingress-dl";
    static final String RABBITMQ_FASTLINE_EXCHANGE = "fastline";
    static final String CONNECTION_FACTORY = "odh-ingress";

    private static Logger LOG = LoggerFactory.getLogger(RabbitMQConnection.class);
    private RabbitMQConfig ingressConfig;

    public RabbitMQConnection() {
        this.ingressConfig = new RabbitMQConfig();
        this.ingressConfig.cluster = ConfigProvider.getConfig().getValue("rabbitmq.cluster", String.class);
        this.ingressConfig.user = ConfigProvider.getConfig().getOptionalValue("rabbitmq.user", String.class);
        this.ingressConfig.pass = ConfigProvider.getConfig().getOptionalValue("rabbitmq.pass", String.class);
        this.ingressConfig.clientName = ConfigProvider.getConfig().getValue("rabbitmq.clientName", String.class);
    }
    
    public ConnectionFactory connectionFactory() {
        String user = this.ingressConfig.user.orElseGet(() -> "*** no user ***");
        String pass = this.ingressConfig.pass.map(p -> "*****").orElseGet(() -> "*** no password ***");

        LOG.info("RabbitMQ cluster: {}", this.ingressConfig.cluster);
        LOG.info("RabbitMQ user: {}", user);
        LOG.info("RabbitMQ password: {}", pass);

        final CachingConnectionFactory fac = new CachingConnectionFactory();
        fac.setConnectionNameStrategy(_f -> ingressConfig.clientName + ": " + System.getenv("HOSTNAME"));
        fac.setAddresses(ingressConfig.cluster);
        if(user != null) {
            fac.setUsername(ingressConfig.user.get());
            fac.setPassword(ingressConfig.pass.get());
        }
        return fac;
    }

    public String getRabbitMQIngressFrom() {
        return String.format("spring-rabbitmq:%s"+
            "?connectionFactory=#bean:%s" +
            "&queues=%s"+
            "&autoDeclare=true"+
            "&acknowledgeMode=AUTO"+
            "&arg.queue.durable=true"+
            "&rejectAndDontRequeue=false"+
            "&exchangePattern=InOnly"+
            "&exchangeType=fanout",
                RABBITMQ_INGRESS_EXCHANGE,
                CONNECTION_FACTORY,
                RABBITMQ_INGRESS_QUEUE);
    }
}
