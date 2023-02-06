// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-bean
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-seda
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-stream
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-paho
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-openapi-java

package it.bz.opendatahub.outbound.fastline;

import org.apache.camel.builder.RouteBuilder;
import org.apache.camel.component.rabbitmq.RabbitMQConstants;

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
import org.apache.camel.component.websocket.WebsocketComponent;

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
    private RabbitMQConfig RabbitMQConfig;

    public FastlineRoute()
    {
        this.RabbitMQConfig = new RabbitMQConfig();
        this.RabbitMQConfig.cluster = ConfigProvider.getConfig().getValue("rabbitmq.cluster", String.class);
        this.RabbitMQConfig.user = ConfigProvider.getConfig().getOptionalValue("rabbitmq.user", String.class);
        this.RabbitMQConfig.pass = ConfigProvider.getConfig().getOptionalValue("rabbitmq.pass", String.class);
    } 

    @Override
    public void configure() {
        String RabbitMQConnectionString = getRabbitMQConnectionString();

        System.out.println(RabbitMQConnectionString);

        // Use RabbitMQ connection
        from(RabbitMQConnectionString)
            .routeId("[Route: fastline]")
            .to("websocket://0.0.0.0:8081/fastline?sendToAll=true");
    }

    private String getRabbitMQConnectionString() {
        final StringBuilder uri = new StringBuilder(String.format("rabbitmq:%s?"+
            "addresses=%s"+
            "&passive=false"+
            "&queue=%s"+
            "&autoAck=false"+
            "&routingKey=#"+ // any routing key
            "&exchangeType=topic"+
            "&skipQueueBind=false"+
            "&autoDelete=false"+
            "&declare=true", 
            "fastline", RabbitMQConfig.cluster, "fastline-q"));

        // Check if RabbitMQ credentials are provided. If so, then add the credentials to the connection string
        RabbitMQConfig.user.ifPresent(user -> uri.append(String.format("&username=%s", user)));
        RabbitMQConfig.pass.ifPresent(pass -> uri.append(String.format("&password=%s", pass)));

        System.out.println(uri.toString());

        return uri.toString();
    }

}