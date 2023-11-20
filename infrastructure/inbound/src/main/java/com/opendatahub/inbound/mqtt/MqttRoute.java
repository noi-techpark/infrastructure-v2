// SPDX-FileCopyrightText: NOI Techpark <digital@noi.bz.it>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// camel-k: dependency=camel:bean
// camel-k: dependency=camel:openapi-java
// camel-k: dependency=camel:paho
// camel-k: dependency=camel:spring-rabbitmq
// camel-k: dependency=camel:seda
// camel-k: dependency=camel:stream
// camel-k: dependency=mvn:org.apache.commons:commons-lang3:3.12.0

package com.opendatahub.inbound.mqtt;

import org.apache.camel.builder.RouteBuilder;
import org.apache.camel.component.paho.PahoConstants;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.amqp.rabbit.connection.CachingConnectionFactory;

import jakarta.enterprise.context.ApplicationScoped;
import java.util.Optional;

import org.eclipse.microprofile.config.ConfigProvider;

import java.time.ZoneId;
import java.time.ZonedDateTime;
import java.time.format.DateTimeFormatter;
import java.util.HashMap;

import org.apache.camel.CamelContext;
import org.apache.camel.Exchange;

import org.apache.camel.component.springrabbit.SpringRabbitMQConstants;
import org.apache.commons.lang3.StringUtils;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.node.ContainerNode;

import java.io.UnsupportedEncodingException;

import java.net.URI;
import java.net.URLDecoder;
import java.net.URISyntaxException;
import java.util.Map;

// import com.opendatahub.RabbitMQConnection;
// import com.opendatahub.WrapperProcessor;

/**
 * MQTT configuration as defined by Quarkus.
 * <p>
 * The data in this interface is taken from System properties, ENV variables,
 * .env file and more. Take a look at https://quarkus.io/guides/config-reference
 * to see how it works.
 */
class MqttConfig {
    String url;

    // Username is optional and may not be set
    Optional<String> user;

    // Password is optional and may not be set
    Optional<String> pass;
}

/**
 * Route to read from MQTT.
 */
@ApplicationScoped
public class MqttRoute extends RouteBuilder {
    private MqttConfig mqttConfig;
    private final RabbitMQConnection rabbitMQConfig;

    public MqttRoute() {
        this.mqttConfig = new MqttConfig();
        this.mqttConfig.url = ConfigProvider.getConfig().getValue("mqtt.url", String.class);
        this.mqttConfig.user = ConfigProvider.getConfig().getOptionalValue("mqtt.user", String.class);
        this.mqttConfig.pass = ConfigProvider.getConfig().getOptionalValue("mqtt.pass", String.class);

        this.rabbitMQConfig = new RabbitMQConnection();
    } 

    @Override
    public void configure() {
        getCamelContext().getRegistry().bind(RabbitMQConnection.CONNECTION_FACTORY, rabbitMQConfig.connectionFactory());

        MqttConfigLogger.log(mqttConfig);

        String mqttConnectionString = getMqttConnectionString();

        // Use MQTT connection
        // wrap message and send to RabbitMQ ingress
        from(mqttConnectionString)
            .routeId("[Route: MQTT subscription]")
            .log("MQTT| ${body}")
            // .log("MQTT| ${headers}")
            .process(exchange -> WrapperProcessor.process(exchange, exchange.getIn().getHeader(PahoConstants.MQTT_TOPIC).toString()))
            .choice()
                // forward to fastline
                .when(header("fastline").isEqualTo(true))
                    // we handle the request as invalid and forward the encapsulated payload to 
                    // whatever mechanism we want to use to store malformed data
                    .to(this.rabbitMQConfig.getRabbitMQFastlineConnectionString())
            .end()
            .choice()
            // if the payload is not a valid json
            .when(header("valid").isEqualTo(false))
                // we handle the request as invalid and forward the encapsulated payload to 
                // whatever mechanism we want to use to store malformed data
                .to(this.rabbitMQConfig.getRabbitMQIngressDeadletterTo())
            .otherwise()
                // otherwise we forward the encapsulated message to the 
                // internal queue waiting to be written in rawDataTable
                .to(this.rabbitMQConfig.getRabbitMQIngressTo())
            .end();
    }

    private String getMqttConnectionString() {
        // paho-mqtt5 has some problems with the retained messages on startup
        // -> https://stackoverflow.com/questions/56248757/camel-paho-routes-not-receiving-offline-messages-while-connecting-back

        // therefore we use paho
        final StringBuilder uri = new StringBuilder(String.format("paho:#?brokerUrl=%s&cleanSession=false&qos=2&clientId=mqtt-route", 
            mqttConfig.url));

        // Check if MQTT credentials are provided. If so, then add the credentials to the connection string
        mqttConfig.user.ifPresent(user -> uri.append(String.format("&userName=%s", user)));
        mqttConfig.pass.ifPresent(pass -> uri.append(String.format("&password=%s", pass)));

        return uri.toString();
    }

}

final class MqttConfigLogger {
    private static Logger LOG = LoggerFactory.getLogger(MqttConfigLogger.class);

    private MqttConfigLogger() {
        // Private constructor, don't allow new instances
    }

    public static void log(MqttConfig config) {
        String url = config.url;
        String user = config.user.orElseGet(() -> "*** no user ***");
        String pass = config.pass.map(p -> "*****").orElseGet(() -> "*** no password ***");

        LOG.info("MQTT URL: {}", url);
        LOG.info("MQTT user: {}", user);
        LOG.info("MQTT password: {}", pass);
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
    Optional<String> clientName;
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
        this.ingressConfig.clientName = ConfigProvider.getConfig().getOptionalValue("rabbitmq.clientName", String.class);
    }
    
    public CachingConnectionFactory connectionFactory() {
        String user = this.ingressConfig.user.orElseGet(() -> "*** no user ***");
        String pass = this.ingressConfig.pass.map(p -> "*****").orElseGet(() -> "*** no password ***");

        LOG.info("RabbitMQ cluster: {}", this.ingressConfig.cluster);
        LOG.info("RabbitMQ user: {}", user);
        LOG.info("RabbitMQ password: {}", pass);
        final CachingConnectionFactory fac = new CachingConnectionFactory();
        fac.setConnectionNameStrategy(_f -> ingressConfig.clientName + ": " + System.getenv("HOSTNAME"));
        fac.setAddresses(ingressConfig.cluster);
        if(user != null) {
            fac.setUsername(user);
            fac.setPassword(pass);
        }
        return fac;
    }

    public String getRabbitMQIngressTo() {
        return String.format("spring-rabbitmq:%s?connectionFactory=#bean:%s&queues=%s&exchangePattern=InOnly&exchangeType=fanout",
                RABBITMQ_INGRESS_EXCHANGE,
                CONNECTION_FACTORY,
                RABBITMQ_INGRESS_QUEUE);
    }

    public String getRabbitMQIngressDeadletterTo() {
        return String.format("spring-rabbitmq:%s?queues=%s&exchangePattern=InOnly&exchangeType=fanout",
                RABBITMQ_INGRESS_DEADLETTER_EXCHANGE,
                RABBITMQ_INGRESS_DEADLETTER_QUEUE);
    }

    public String getRabbitMQFastlineConnectionString() {
        final StringBuilder uri = new StringBuilder(String.format("spring-rabbitmq:%s?exchangePattern=InOnly&exchangeType=topic",
                RABBITMQ_FASTLINE_EXCHANGE));
        return uri.toString();
    }
}

class WrapperProcessor {
    public static void process(final Exchange exchange, final String provider) throws JsonProcessingException {
        HashMap<String, Object> map = new HashMap<String, Object>();
        ObjectMapper objectMapper = new ObjectMapper();

        String payload = exchange.getIn().getBody(String.class);

        map.put("rawdata", payload);
        map.put("timestamp", ZonedDateTime.now(ZoneId.of("UTC")).format(DateTimeFormatter.ISO_INSTANT));

        // We start encapsulating the payload in a new message where we have
        // {provider: ..., timestamp: ..., rawdata: ...}
        // timestamp indicates when we received the message
        // provider is the provided which sent the message
        // rawdata is the data sent

        // provider has the same format as any URI.
        // it might specify query params to request some special behaviour
        // EG: mobility/tourism?fastline=true
        // EG: 'provider/collection/...&params'
        URI providerURI = null;
        Map<String, String> query = null;
        Boolean validProvider = true;
        try {
            providerURI = new URI(StringUtils.strip(provider, "/"));
            query = parseQuerystring(providerURI.getQuery());
        } catch (URISyntaxException e) {
            validProvider = false;
        }

        if (!validProvider || null == providerURI.getPath()) {
            System.out.println("invalid provider: "+ provider);

            // invalid provider, therefore we put the raw provider and send the message to the deadletter
            map.put("provider", provider);
            exchange.getMessage().setBody(objectMapper.writeValueAsString(map));
            exchange.getMessage().setHeader("valid", false);
            return;
        }

        // setting up provider routeKey
        String routeKey = providerURI.getPath().replaceAll("/", ".");
        System.out.println("routing to routeKey " +  routeKey);
        System.out.println("provider " +  provider);

        //https://github.com/Talend/apache-camel/blob/master/components/camel-rabbitmq/src/main/java/org/apache/camel/component/rabbitmq/SpringRabbitMQConstants.java
        exchange.getMessage().setHeader(SpringRabbitMQConstants.ROUTING_KEY, routeKey);
        //exchange.getMessage().setHeader(SpringRabbitMQConstants.RABBITMQ_DEAD_LETTER_ROUTING_KEY, routeKey);

        // if the provider specifies the fastline=true param
        // set the header
        if (query.containsKey("fastline") && query.get("fastline").equals("true")) {
            exchange.getMessage().setHeader("fastline", true);
            System.out.println("is fastline!");
        }

        map.put("provider", providerURI.toString());
        exchange.getMessage().setBody(objectMapper.writeValueAsString(map));

        if (isValidJSON(payload)) {
            exchange.getMessage().setHeader("valid", true);
        } else {
            exchange.getMessage().setHeader("valid", false);
        }
    }

    static public boolean isValidJSON(final String json) {
        try {
            final ObjectMapper objectMapper = new ObjectMapper();
            final JsonNode jsonNode = objectMapper.readTree(json);
            return jsonNode instanceof ContainerNode;
        } catch (Exception jpe) {
            return false;
        }
    }

    /**
     * Parse a querystring into a map of key/value pairs.
     *
     * @param queryString the string to parse (without the '?')
     * @return key/value pairs mapping to the items in the querystring
     */
    public static Map<String, String> parseQuerystring(String queryString) {
        Map<String, String> map = new HashMap<String, String>();
        if ((queryString == null) || (queryString.equals(""))) {
            return map;
        }
        String[] params = queryString.split("&");
        for (String param : params) {
            try {
                String[] keyValuePair = param.split("=", 2);
                String name = URLDecoder.decode(keyValuePair[0], "UTF-8");
                if (name == "") {
                    continue;
                }
                String value = keyValuePair.length > 1 ? URLDecoder.decode(
                        keyValuePair[1], "UTF-8") : "";
                map.put(name, value);
            } catch (UnsupportedEncodingException e) {
                // ignore this parameter if it can't be decoded
            }
        }
        return map;
    }
}
