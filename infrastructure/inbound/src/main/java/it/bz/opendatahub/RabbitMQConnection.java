// SPDX-FileCopyrightText: NOI Techpark <digital@noi.bz.it>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package it.bz.opendatahub;

import java.util.Optional;
import org.eclipse.microprofile.config.ConfigProvider;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;


class RabbitMQConfig {
    String cluster;

    // Username is optional and may not be set
    Optional<String> user;

    // Password is optional and may not be set
    Optional<String> pass;
}

public class RabbitMQConnection {

    static final String RABBITMQ_INGRESS_QUEUE = "ingress-q";
    static final String RABBITMQ_INGRESS_EXCHANGE = "ingress";
    static final String RABBITMQ_INGRESS_DEADLETTER_QUEUE = "ingress-dl-q";
    static final String RABBITMQ_INGRESS_DEADLETTER_EXCHANGE = "ingress-dl";
    static final String RABBITMQ_FASTLINE_EXCHANGE = "fastline";

    private static Logger LOG = LoggerFactory.getLogger(RabbitMQConnection.class);
    private RabbitMQConfig ingressConfig;

    public RabbitMQConnection() {
        this.ingressConfig = new RabbitMQConfig();
        this.ingressConfig.cluster = ConfigProvider.getConfig().getValue("rabbitmq.cluster", String.class);
        this.ingressConfig.user = ConfigProvider.getConfig().getOptionalValue("rabbitmq.user", String.class);
        this.ingressConfig.pass = ConfigProvider.getConfig().getOptionalValue("rabbitmq.pass", String.class);

        String user = this.ingressConfig.user.orElseGet(() -> "*** no user ***");
        String pass = this.ingressConfig.pass.map(p -> "*****").orElseGet(() -> "*** no password ***");

        LOG.info("RabbitMQ cluster: {}", this.ingressConfig.cluster);
        LOG.info("RabbitMQ user: {}", user);
        LOG.info("RabbitMQ password: {}", pass);
    }

    private String setAuth(StringBuilder uri) {
        // Check if RabbitMQ credentials are provided. If so, then add the credentials to the connection string
        this.ingressConfig.user.ifPresent(user -> uri.append(String.format("&username=%s", user)));
        this.ingressConfig.pass.ifPresent(pass -> uri.append(String.format("&password=%s", pass)));

        System.out.println(uri.toString());
        return uri.toString();
    }

    public String getRabbitMQIngressFrom() {
        final StringBuilder uri = new StringBuilder(String.format("rabbitmq:?"+
            "addresses=%s"+
            "&queue=%s"+
            "&autoAck=false"+
            // setting reQueue=true + autoAck=false messages not processed because of exceptions get requeued
            "&reQueue=true"+ 
            "&autoDelete=false"+
            "&skipExchangeDeclare=true"+
            "&skipQueueBind=true"+
            "&skipQueueDeclare=true",
            this.ingressConfig.cluster,
            RABBITMQ_INGRESS_QUEUE));

        return this.setAuth(uri);
    }

    public String getRabbitMQIngressTo() {
        final StringBuilder uri = new StringBuilder(String.format("rabbitmq:%s?"+
            "addresses=%s"+
            "&queue=%s"+
            "&autoDelete=false"+
            // https://stackoverflow.com/questions/14527185/activemq-i-cant-consume-a-message-sent-from-camel-using-inout-pattern
            // https://camel.apache.org/manual/exchange-pattern.html
            // we are using Event messages, therefore we have to specify the InOnly pattern
            // otherwise the component expects a reply
            "&exchangePattern=InOnly"+
            "&exchangeType=fanout",
            RABBITMQ_INGRESS_EXCHANGE,
            this.ingressConfig.cluster,
            RABBITMQ_INGRESS_QUEUE));

        return this.setAuth(uri);
    }

    public String getRabbitMQIngressDeadletterTo() {
        final StringBuilder uri = new StringBuilder(String.format("rabbitmq:%s?"+
            "addresses=%s"+
            "&queue=%s"+
            "&routingKey=ingress.*"+
            "&exchangeType=fanout"+
            "&exchangePattern=InOnly"+
            "&autoDelete=false", 
            RABBITMQ_INGRESS_DEADLETTER_EXCHANGE,
            this.ingressConfig.cluster,
            RABBITMQ_INGRESS_DEADLETTER_QUEUE));

        return this.setAuth(uri);
    }

    public String getRabbitMQFastlineConnectionString() {
        final StringBuilder uri = new StringBuilder(String.format("rabbitmq:%s?"+
            "addresses=%s"+
            "&passive=true"+
            "&exchangeType=topic"+
            "&skipQueueBind=true"+
            "&skipQueueDeclare=true"+
            "&exchangePattern=InOnly"+
            "&autoDelete=false"+
            "&declare=true", 
            RABBITMQ_FASTLINE_EXCHANGE, this.ingressConfig.cluster));

        return this.setAuth(uri);
    }
}
