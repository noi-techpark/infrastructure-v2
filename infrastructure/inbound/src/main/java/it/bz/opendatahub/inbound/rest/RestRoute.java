// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-bean
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-seda
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-stream
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-openapi-java
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-rest
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-paho

package it.bz.opendatahub.inbound.rest;

import it.bz.opendatahub.RabbitMQConnection;
import org.apache.camel.component.rabbitmq.RabbitMQConstants;

import org.apache.camel.Exchange;
import org.apache.camel.builder.RouteBuilder;
import org.apache.camel.model.rest.RestBindingMode;

import com.fasterxml.jackson.databind.*;
import com.fasterxml.jackson.databind.node.ContainerNode;

import javax.enterprise.context.ApplicationScoped;

import java.time.ZoneId;
import java.time.ZonedDateTime;
import java.time.format.DateTimeFormatter;

import java.util.HashMap;
import java.util.Map;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

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
        // Exposes REST connection
        // process and forward to the internal queue waiting to be written in rawDataTable
        restConfiguration()
            .apiContextPath("/api-doc")
            .apiProperty("api.title", "ODH inbound REST API")
            .apiProperty("api.version", "0.0.1")
            .bindingMode(RestBindingMode.auto)
            .contextPath("/")
            .host("0.0.0.0")
            .port(8080);
            

        from("rest:post:/{provider}")
            .process(exchange -> {
                Map<String, Object> map = new HashMap<String, Object>();
                ObjectMapper objectMapper = new ObjectMapper();

                String payload = exchange.getIn().getBody(String.class);
                // We start encapsulating the payload in a new message where we have
                // {provider: ..., timestamp: ..., rawdata: ...}
                // timestamp indicates when we received the message
                // provider is the provided which sent the message
                // rawdata is the data sent

                // provider is populated using the uri path of the request (request on /flightdata -> flightdat)
                // we might use a proper function to transform the request path into provider

                String provider = exchange.getIn().getHeader("provider").toString();
                String routeKey = String.format("ingress.%s", provider);

                map.put("provider", provider);
                map.put("rawdata", payload);
                map.put("timestamp", ZonedDateTime.now(ZoneId.of("UTC")).format(DateTimeFormatter.ISO_INSTANT));
                
                exchange.getMessage().setBody(objectMapper.writeValueAsString(map));
                //https://github.com/Talend/apache-camel/blob/master/components/camel-rabbitmq/src/main/java/org/apache/camel/component/rabbitmq/RabbitMQConstants.java
                exchange.getMessage().setHeader(RabbitMQConstants.ROUTING_KEY, routeKey);
                exchange.getMessage().setHeader(RabbitMQConstants.RABBITMQ_DEAD_LETTER_ROUTING_KEY, routeKey);
                
                if (isValidJSON(payload)) {
                    exchange.getMessage().setHeader("validPayload", true);
                } else {
                    exchange.getMessage().setHeader("validPayload", false);
                }
            })
            .log("REST| ${body}")
            // .log("REST| ${headers}")
            .choice()
                // if the payload is not a valid json
            .when(header("validPayload").isEqualTo(false))
                // we handle the request as invalid and forward the encapsulated payload to 
                // whatever mechanism we want to use to store malformed data
                .to(this.rabbitMQConfig.getRabbitMQIngressDeadletterConnectionString())
            .otherwise()
                // otherwise we forward the encapsulated message to the 
                // internal queue waiting to be written in rawDataTable
                .to(this.rabbitMQConfig.getRabbitMQIngressConnectionString())
            .end()

            // reset and send responses
            .setHeader(Exchange.HTTP_RESPONSE_CODE, constant(200))
            .setBody(constant(null))
            /*.endRest()*/;
    }

    public boolean isValidJSON(final String json) {
        try {
            final ObjectMapper objectMapper = new ObjectMapper();
            final JsonNode jsonNode = objectMapper.readTree(json);
            return jsonNode instanceof ContainerNode;
        } catch (Exception jpe) {
            return false;
        }
    }
}

final class RestConfigLogger {

    private static Logger LOG = LoggerFactory.getLogger(RestConfigLogger.class);

    private RestConfigLogger() {
        // Private constructor, don't allow new instances
    }
}