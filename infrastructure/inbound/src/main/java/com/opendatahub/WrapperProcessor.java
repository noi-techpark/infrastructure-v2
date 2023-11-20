// SPDX-FileCopyrightText: NOI Techpark <digital@noi.bz.it>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package com.opendatahub;

import java.time.ZoneId;
import java.time.ZonedDateTime;
import java.time.format.DateTimeFormatter;
import java.util.HashMap;

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

public class WrapperProcessor {
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
