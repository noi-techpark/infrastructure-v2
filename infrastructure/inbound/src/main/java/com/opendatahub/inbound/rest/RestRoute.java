// SPDX-FileCopyrightText: NOI Techpark <digital@noi.bz.it>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// camel-k: dependency=camel:bean
// camel-k: dependency=camel:openapi-java
// camel-k: dependency=camel:paho
// camel-k: dependency=camel:rest
// camel-k: dependency=camel:spring-rabbitmq
// camel-k: dependency=camel:seda
// camel-k: dependency=camel:stream

package com.opendatahub.inbound.rest;

// import com.opendatahub.RabbitMQConnection;
// import com.opendatahub.WrapperProcessor;

import org.apache.camel.Exchange;
import org.apache.camel.builder.RouteBuilder;
import org.apache.camel.model.rest.RestBindingMode;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.amqp.rabbit.connection.CachingConnectionFactory;
import org.springframework.amqp.rabbit.connection.ConnectionFactory;

import jakarta.enterprise.context.ApplicationScoped;
import jakarta.inject.Named;

import java.util.Optional;

import org.eclipse.microprofile.config.ConfigProvider;

import java.time.ZoneId;
import java.time.ZonedDateTime;
import java.time.format.DateTimeFormatter;
import java.util.HashMap;

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

/**
 * Route to read from REST.
 */
@ApplicationScoped
public class RestRoute extends RouteBuilder {
    private final RabbitMQConnection rabbitMQConfig;

    public RestRoute()
    {
        this.rabbitMQConfig = new RabbitMQConnection();
    } 

    @Override
    public void configure() {
        getCamelContext().getRegistry().bind(RabbitMQConnection.CONNECTION_FACTORY, rabbitMQConfig.connectionFactory());
        // Exposes REST connection
        // wrap message and send to RabbitMQ ingress
        restConfiguration()
            .apiContextPath("/api-doc")
            .apiProperty("api.title", "ODH inbound REST API")
            .apiProperty("api.version", "0.0.1")
            .bindingMode(RestBindingMode.auto)
            .contextPath("/")
            .host("0.0.0.0")
            .port(8080);
            

        from("rest:post:/*")
            // rest query params are passed as headers
            // therefore we have to delete reserved header names
            .removeHeaders("fastline")
            .removeHeaders("valid")

            .process(exchange -> WrapperProcessor.process(exchange, 
                exchange.getIn().getHeader(Exchange.HTTP_URI).toString()))
            // .log("REST| ${body}")
            // .log("REST| ${headers}")
            // we clear all CamelHttp since they could create problem
            // with other components
            .removeHeaders("CamelHttp*")
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
            .end()
            // reset and send responses
            .setBody(constant(null))
            .removeHeaders("*")
            .setHeader(Exchange.HTTP_RESPONSE_CODE, constant(200));
    }
}

final class RestConfigLogger {

    private static Logger LOG = LoggerFactory.getLogger(RestConfigLogger.class);

    private RestConfigLogger() {
        // Private constructor, don't allow new instances
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
        fac.setPort(0);
        if(user != null) {
            fac.setUsername(ingressConfig.user.get());
            fac.setPassword(ingressConfig.pass.get());
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
