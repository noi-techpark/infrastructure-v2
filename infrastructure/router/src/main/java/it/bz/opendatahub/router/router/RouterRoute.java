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

import com.fasterxml.jackson.databind.*;
import com.fasterxml.jackson.databind.node.ContainerNode;

import java.util.Optional;

import org.eclipse.microprofile.config.ConfigProvider;


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
                .log("RabbitMQ| ${headers}");
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
                    // we then compute the database connection usong the message body (in this case we only care bout the field `provider`)
                    // and store the connection string in the `database` header to be used later
                    exchange.getMessage().setHeader("database", getDatabaseString((String)body.get("provider")));
                })
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