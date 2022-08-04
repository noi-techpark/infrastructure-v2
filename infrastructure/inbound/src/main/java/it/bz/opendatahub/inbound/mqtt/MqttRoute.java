// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-bean
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-seda
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-stream
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-paho-mqtt5
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-openapi-java

package it.bz.opendatahub.inbound.mqtt;

import org.apache.camel.builder.RouteBuilder;
import org.apache.camel.component.paho.mqtt5.PahoMqtt5Constants;

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
 * MQTT configuration as defined by Quarkus.
 * <p>
 * The data in this interface is taken from System properties, ENV variables,
 * .env file and more. Take a look at https://quarkus.io/guides/config-reference
 * to see how it works.
 */
class MqttConfig {
    public static final String SEDA_MQTT_QUEUE_OUT = "seda:queue_out?multipleConsumers=true";

    String url;

    // Username is optional and may not be set
    Optional<String> user;

    // Password is optional and may not be set
    Optional<String> pass;

    public String storage_url;
    public String storage_topic;

    public Optional<String> storage_user;
    public Optional<String> storage_password;
}

/**
 * Route to read from MQTT.
 */
@ApplicationScoped
public class MqttRoute extends RouteBuilder {
    private MqttConfig mqttConfig;

    public MqttRoute()
    {
        this.mqttConfig = new MqttConfig();
        this.mqttConfig.url = ConfigProvider.getConfig().getValue("mqtt.url", String.class);
        this.mqttConfig.user = ConfigProvider.getConfig().getOptionalValue("mqtt.user", String.class);
        this.mqttConfig.pass = ConfigProvider.getConfig().getOptionalValue("mqtt.pass", String.class);

        this.mqttConfig.storage_url = ConfigProvider.getConfig().getValue("internal_mqtt.url", String.class);
        this.mqttConfig.storage_topic = ConfigProvider.getConfig().getValue("internal_mqtt.topic", String.class);
        this.mqttConfig.storage_user = ConfigProvider.getConfig().getOptionalValue("internal_mqtt.user", String.class);
        this.mqttConfig.storage_password = ConfigProvider.getConfig().getOptionalValue("internal_mqtt.password", String.class);
    } 

    @Override
    public void configure() {
        MqttConfigLogger.log(mqttConfig);

        String mqttConnectionString = getMqttConnectionString();

        // Use MQTT connection
        // process and forward to SEDA_MQTT_QUEUE_OUT
        from(mqttConnectionString)
                .routeId("[Route: MQTT subscription]")
                .setHeader("topic", header(PahoMqtt5Constants.MQTT_TOPIC))
                // Put body as string into header "rawdata" for later reuse
                .process(exchange -> {
                    Map<String, Object> map = new HashMap<String, Object>();
                    ObjectMapper objectMapper = new ObjectMapper();
                    
                    String payload = exchange.getMessage().getBody(String.class);
                    map.put("provider", exchange.getMessage().getHeader("topic").toString());
                    map.put("rawdata", payload);
                    map.put("timestamp", ZonedDateTime.now(ZoneId.of("UTC")).format(DateTimeFormatter.ISO_INSTANT));
                    exchange.getMessage().setHeader("provider", exchange.getMessage().getHeader("topic").toString());
                    exchange.getMessage().setBody(objectMapper.writeValueAsString(map));
                        
                    if (isValidJSON(payload)) {
                        exchange.getMessage().setHeader("validPayload", true);
                    } else {
                        exchange.getMessage().setHeader("validPayload", false);
                    }
                    
                })
                .log("MQTT| ${body}")
                .log("MQTT| ${headers}")
                .choice()
                    .when(header("validPayload").isEqualTo(false))
                    .log("ERROR NOT A VALID PAYLOAD, ROUTE TO FALIED STORAGE")
                .otherwise()
                    .to(getInternalStorageQueueConnectionString())
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

    private String getMqttConnectionString() {
        final StringBuilder uri = new StringBuilder(String.format("paho-mqtt5:#?brokerUrl=%s&cleanStart=false&qos=2&clientId=mqtt-route", 
            mqttConfig.url));

        // Check if MQTT credentials are provided. If so, then add the credentials to the connection string
        mqttConfig.user.ifPresent(user -> uri.append(String.format("&userName=%s", user)));
        mqttConfig.pass.ifPresent(pass -> uri.append(String.format("&password=%s", pass)));

        return uri.toString();
    }

    private String getInternalStorageQueueConnectionString() {
        // TODO use AmazonSNS uri if needed
        // for testing purpose we use Mosquitto
        final StringBuilder uri = new StringBuilder(String.format("paho-mqtt5:%s?brokerUrl=%s&qos=2", 
        mqttConfig.storage_topic, mqttConfig.storage_url));

        // Check if MQTT credentials are provided. If so, then add the credentials to the connection string
        mqttConfig.storage_user.ifPresent(user -> uri.append(String.format("&userName=%s", user)));
        mqttConfig.storage_password.ifPresent(pass -> uri.append(String.format("&password=%s", pass)));

        return uri.toString();
    }
}

/**
 * Utility class to log {@link MqttConfig}.
 */
final class MqttConfigLogger {

    private static Logger LOG = LoggerFactory.getLogger(MqttConfigLogger.class);

    private MqttConfigLogger() {
        // Private constructor, don't allow new instances
    }

    /**
     * Log {@link MqttConfig}.
     *
     * @param config The {@link MqttConfig} to log.
     */
    public static void log(MqttConfig config) {
        String url = config.url;
        String user = config.user.orElseGet(() -> "*** no user ***");
        String pass = config.pass.map(p -> "*****").orElseGet(() -> "*** no password ***");

        LOG.info("MQTT URL: {}", url);
        LOG.info("MQTT user: {}", user);
        LOG.info("MQTT password: {}", pass);

        LOG.info("INTARNAL MQTT URL: {}", config.storage_url);
        LOG.info("INTARNAL MQTT user: {}", config.storage_topic);
    }
}