// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-bean
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-seda
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-stream
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-paho
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-openapi-java
// camel-k: dependency=mvn:org.apache.commons:commons-lang3:3.12.0

package it.bz.opendatahub.inbound.mqtt;

import org.apache.camel.builder.RouteBuilder;
import org.apache.camel.component.paho.PahoConstants;

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
import org.apache.commons.lang3.StringUtils;

import org.eclipse.microprofile.config.ConfigProvider;

import it.bz.opendatahub.RabbitMQConnection;
import org.apache.camel.component.rabbitmq.RabbitMQConstants;

import org.apache.camel.component.jackson.JacksonDataFormat;

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

    public Optional<String> storage_user;
    public Optional<String> storage_password;
}

/**
 * Route to read from MQTT.
 */
@ApplicationScoped
public class MqttRoute extends RouteBuilder {
    private MqttConfig mqttConfig;
    private final RabbitMQConnection rabbitMQConfig;

    // TODO the topic should mimic an url 
    // like 'provider/collection/...&params'
    // the path will be used to compute db/collection where to store data & the routing key
    // for the transformer
    // while params can be used to annotate the messages like add &fastline=true
    // to enable fastline broadcast
    // queryParam parser: http://www.java2s.com/example/java-utility-method/url-query-parse/parsequerystring-string-querystring-86933.html
    // url parser: https://docs.oracle.com/javase/tutorial/networking/urls/urlInfo.html
    public MqttRoute()
    {
        this.mqttConfig = new MqttConfig();
        this.mqttConfig.url = ConfigProvider.getConfig().getValue("mqtt.url", String.class);
        this.mqttConfig.user = ConfigProvider.getConfig().getOptionalValue("mqtt.user", String.class);
        this.mqttConfig.pass = ConfigProvider.getConfig().getOptionalValue("mqtt.pass", String.class);

        this.rabbitMQConfig = new RabbitMQConnection();
    } 

    @Override
    public void configure() {
        MqttConfigLogger.log(mqttConfig);

        String mqttConnectionString = getMqttConnectionString();

        // Use MQTT connection
        // process and forward to the internal queue waiting to be written in rawDataTable
        // TODO Add throtling if needed
        // TODO If error occurs, don't ACK message
        from(mqttConnectionString)
            .routeId("[Route: MQTT subscription]")
            .log("MQTT| ${body}")
            // .log("MQTT| ${headers}")
            .process(exchange -> {
                Map<String, Object> map = new HashMap<String, Object>();
                ObjectMapper objectMapper = new ObjectMapper();
                
                String payload = exchange.getMessage().getBody(String.class);

                // We start encapsulating the payload in a new message where we have
                // {provider: ..., timestamp: ..., rawdata: ...}
                // timestamp indicates when we received the message
                // provider is the provided which sent the message
                // rawdata is the data sent

                // provider is populated using the topic `exchange.getMessage().getHeader(PahoConstants.MQTT_TOPIC).toString()`
                // we might use a proper function to remove the first "/" and normalize subpaths (/open/test -> open_test)

                String provider = exchange.getIn().getHeader(PahoConstants.MQTT_TOPIC).toString();
                provider = StringUtils.stripStart(provider, "/");
                String routeKey = String.format("ingress.%s", provider);

                map.put("provider", provider);
                map.put("rawdata", payload);
                map.put("timestamp", ZonedDateTime.now(ZoneId.of("UTC")).format(DateTimeFormatter.ISO_INSTANT));

                exchange.getMessage().setHeader("provider", provider);
                exchange.getMessage().setBody(objectMapper.writeValueAsString(map));

                //https://github.com/Talend/apache-camel/blob/master/components/camel-rabbitmq/src/main/java/org/apache/camel/component/rabbitmq/RabbitMQConstants.java
                exchange.getMessage().setHeader(RabbitMQConstants.ROUTING_KEY, routeKey);
                exchange.getMessage().setHeader(RabbitMQConstants.RABBITMQ_DEAD_LETTER_ROUTING_KEY, routeKey);
                    
                // We validate the payload checking it is a proper json
                if (isValidJSON(payload)) {
                    exchange.getMessage().setHeader("validPayload", true);
                    ObjectMapper mapper = new ObjectMapper();
                    Map<String, Object> body = (HashMap<String, Object>)mapper.readValue(payload, Map.class);
                   
                    Object fastline = body.get("fastline");
                    if (null != fastline && fastline instanceof Boolean && (Boolean)fastline == true) {
                        exchange.getMessage().setHeader("fastline", true);
                    }
                } else {
                    exchange.getMessage().setHeader("validPayload", false);
                }
                
            })
            .choice()
            // forward to fastline
            .when(header("fastline").isEqualTo(true))
                // we handle the request as invalid and forward the encapsulated payload to 
                // whatever mechanism we want to use to store malformed data
                .to(this.rabbitMQConfig.getRabbitMQFastlineConnectionString())
            .end()
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
            .end();
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

    // When using Mosquitto
    //      Connecting to the perimetral MQTT is not trivial and both publishers and subscribers follow a certain
    //      agreement to ensure no message will be lost:
    //      Publishers MUST publish with QoS >= 1
    //      Subscribers MUST connect with QoS >= 1
    //      Subscribers MUST connect with cleanStart = false
    //      ALL Subscribers in ALL pods must connect with a unique clientId which can't change at pod restart
    //      Read https://www.hivemq.com/blog/mqtt-essentials-part-7-persistent-session-queuing-messages/
    //      https://stackoverflow.com/questions/52439954/get-all-messages-after-the-client-has-re-connected-to-the-mqtt-broker
    //      to know more aboutn Persistent COnnection 
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