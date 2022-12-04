// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-bean
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-seda
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-stream
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-paho
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-openapi-java

package it.bz.opendatahub.inbound.router;

import org.apache.camel.builder.RouteBuilder;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import javax.enterprise.context.ApplicationScoped;

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

    String ingressQueue;

    // Username is optional and may not be set
    Optional<String> user;

    // Password is optional and may not be set
    Optional<String> pass;
}

class Payload {
    public String test;
    public Integer lal;

}

/**
 * Route to read from RabbitMQ.
 */
@ApplicationScoped
public class RouterRoute extends RouteBuilder {
    private RabbitMQConfig RabbitMQConfig;

    public RouterRoute()
    {
        this.RabbitMQConfig = new RabbitMQConfig();
        this.RabbitMQConfig.cluster = ConfigProvider.getConfig().getValue("rabbitmq.cluster", String.class);
        this.RabbitMQConfig.ingressQueue = ConfigProvider.getConfig().getValue("rabbitmq.ingress-queue", String.class);
        this.RabbitMQConfig.user = ConfigProvider.getConfig().getOptionalValue("rabbitmq.user", String.class);
        this.RabbitMQConfig.pass = ConfigProvider.getConfig().getOptionalValue("rabbitmq.pass", String.class);
    } 

    @Override
    public void configure() {
        RabbitMQConfigLogger.log(RabbitMQConfig);

        String RabbitMQConnectionString = getRabbitMQConnectionString();

        System.out.println(RabbitMQConnectionString);

        // Use RabbitMQ connection
        from(RabbitMQConnectionString)
                .routeId("[Route: RabbitMQ subscription]")
                .log("RabbitMQ| ${body}")
                .log("RabbitMQ| ${headers}")
                .unmarshal().json(JsonLibrary.Jackson, Payload.class)
                .log("RabbitMQ| ${body}")
                .process(exchange -> {
                    System.out.println(exchange.getMessage().getBody());

                    // First we unmarshal the payload
                    //Map<String, Object> body = (HashMap<String, Object>)exchange.getMessage().getBody(Map.class);
                    //Object timestamp = body.get("timestamp");
                });
    }

    private String getRabbitMQConnectionString() {
        final StringBuilder uri = new StringBuilder(String.format("rabbitmq:?"+
            "addresses=%s"+
            "&passive=true"+
            "&queue=%s"+
            "&autoAck=false"+
            "&skipExchangeDeclare=true"+
            "&skipQueueBind=true"+
            "&autoDelete=false"+
            "&declare=false", 
            RabbitMQConfig.cluster, RabbitMQConfig.ingressQueue));

        // Check if RabbitMQ credentials are provided. If so, then add the credentials to the connection string
        RabbitMQConfig.user.ifPresent(user -> uri.append(String.format("&username=%s", user)));
        RabbitMQConfig.pass.ifPresent(pass -> uri.append(String.format("&password=%s", pass)));

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
        LOG.info("RabbitMQ ingressQueue: {}", config.ingressQueue);
        LOG.info("RabbitMQ user: {}", user);
        LOG.info("RabbitMQ password: {}", pass);
    }
}