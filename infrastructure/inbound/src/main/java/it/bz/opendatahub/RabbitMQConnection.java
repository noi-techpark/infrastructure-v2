package it.bz.opendatahub;

import java.util.Optional;
import org.eclipse.microprofile.config.ConfigProvider;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;


class RabbitMQConfig {
    String cluster;

    String ingressQueue;
    String ingressTopic;

    String ingressDLQueue;
    String ingressDLTopic;

    // Username is optional and may not be set
    Optional<String> user;

    // Password is optional and may not be set
    Optional<String> pass;
}

public class RabbitMQConnection {

    private static Logger LOG = LoggerFactory.getLogger(RabbitMQConnection.class);
    private RabbitMQConfig ingressConfig;

    public RabbitMQConnection() {
        this.ingressConfig = new RabbitMQConfig();
        this.ingressConfig.cluster = ConfigProvider.getConfig().getValue("rabbitmq.cluster", String.class);
        this.ingressConfig.ingressQueue = ConfigProvider.getConfig().getValue("rabbitmq.ingress-queue", String.class);
        this.ingressConfig.ingressTopic = ConfigProvider.getConfig().getValue("rabbitmq.ingress-topic", String.class);
        this.ingressConfig.ingressDLQueue = ConfigProvider.getConfig().getValue("rabbitmq.ingress-dl-queue", String.class);
        this.ingressConfig.ingressDLTopic = ConfigProvider.getConfig().getValue("rabbitmq.ingress-dl-topic", String.class);
        this.ingressConfig.user = ConfigProvider.getConfig().getOptionalValue("rabbitmq.user", String.class);
        this.ingressConfig.pass = ConfigProvider.getConfig().getOptionalValue("rabbitmq.pass", String.class);

        String user = this.ingressConfig.user.orElseGet(() -> "*** no user ***");
        String pass = this.ingressConfig.pass.map(p -> "*****").orElseGet(() -> "*** no password ***");

        LOG.info("RabbitMQ cluster: {}", this.ingressConfig.cluster);
        LOG.info("RabbitMQ ingressTopic: {}", this.ingressConfig.ingressTopic);
        LOG.info("RabbitMQ ingressQueue: {}", this.ingressConfig.ingressQueue);
        LOG.info("RabbitMQ ingressDLTopic: {}", this.ingressConfig.ingressDLTopic);
        LOG.info("RabbitMQ ingressDLQueue: {}", this.ingressConfig.ingressDLQueue);
        LOG.info("RabbitMQ user: {}", user);
        LOG.info("RabbitMQ password: {}", pass);
    }

    public String getRabbitMQIngressConnectionString() {
        final StringBuilder uri = new StringBuilder(String.format("rabbitmq:%s?"+
            "addresses=%s"+
            "&queue=%s"+
            "&autoAck=false"+
            // setting reQueue=true + autoAck=false messages not processed because of exceptions get requeued
            "&reQueue=true"+ 
            "&routingKey=ingress.*"+
            // "&deadLetterExchange=%s"+
            "&passive=false"+
            // "&skipExchangeDeclare=true"+
            // "&skipQueueBind=true"+
            "&autoDelete=false"+
            "&publisherAcknowledgements=false"+
            "&exchangeType=topic"+
            // https://stackoverflow.com/questions/14527185/activemq-i-cant-consume-a-message-sent-from-camel-using-inout-pattern
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

    public String getRabbitMQIngressDeadletterConnectionString() {
        final StringBuilder uri = new StringBuilder(String.format("rabbitmq:%s?"+
            "addresses=%s"+
            // "&passive=true"+
            "&queue=%s"+
            "&routingKey=ingress.*"+
            "&autoAck=false"+
            // "&skipExchangeDeclare=true"+
            // "&skipQueueBind=true"+
            "&exchangeType=topic"+
            "&exchangePattern=InOnly"+
            "&autoDelete=false"+
            "&declare=true", 
            this.ingressConfig.ingressDLTopic,
            this.ingressConfig.cluster,
            this.ingressConfig.ingressDLQueue));

        // Check if RabbitMQ credentials are provided. If so, then add the credentials to the connection string
        this.ingressConfig.user.ifPresent(user -> uri.append(String.format("&username=%s", user)));
        this.ingressConfig.pass.ifPresent(pass -> uri.append(String.format("&password=%s", pass)));

        return uri.toString();
    }

    public String getRabbitMQFastlineConnectionString() {
        final StringBuilder uri = new StringBuilder(String.format("rabbitmq:%s?"+
            "addresses=%s"+
            // "&passive=true"+
            "&autoAck=false"+
            // "&skipExchangeDeclare=true"+
            "&skipQueueBind=true"+
            "&exchangeType=topic"+
            "&exchangePattern=InOnly"+
            "&autoDelete=false"+
            "&declare=true", 
            "fastline",
            this.ingressConfig.cluster));

        // Check if RabbitMQ credentials are provided. If so, then add the credentials to the connection string
        this.ingressConfig.user.ifPresent(user -> uri.append(String.format("&username=%s", user)));
        this.ingressConfig.pass.ifPresent(pass -> uri.append(String.format("&password=%s", pass)));

        return uri.toString();
    }
}
