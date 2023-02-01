package it.bz.opendatahub;

import java.time.ZoneId;
import java.time.ZonedDateTime;
import java.time.format.DateTimeFormatter;
import java.util.HashMap;

import org.apache.camel.Exchange;
import org.apache.camel.component.rabbitmq.RabbitMQConstants;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.node.ContainerNode;

public class WrapperProcessor {
    public static void process(final Exchange exchange, final String provider) throws JsonProcessingException {
        HashMap<String, Object> map = new HashMap<String, Object>();
        ObjectMapper objectMapper = new ObjectMapper();

        String payload = exchange.getIn().getBody(String.class);
        // We start encapsulating the payload in a new message where we have
        // {provider: ..., timestamp: ..., rawdata: ...}
        // timestamp indicates when we received the message
        // provider is the provided which sent the message
        // rawdata is the data sent

        // provider is populated using the uri path of the request (request on /flightdata -> flightdat)
        // we might use a proper function to transform the request path into provider

        String routeKey = String.format("ingress.%s", provider);
        System.out.println("routing to routeKey " +  routeKey);

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
}
