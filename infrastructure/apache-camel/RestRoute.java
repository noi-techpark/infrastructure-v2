// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-bean
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-seda
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-stream
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-openapi-java
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-rest
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-paho-mqtt5

package it.bz.opendatahub.inbound.rest;

import org.apache.camel.Exchange;
import org.apache.camel.builder.RouteBuilder;
import org.apache.camel.model.rest.RestBindingMode;

import com.fasterxml.jackson.databind.ObjectMapper;

import javax.enterprise.context.ApplicationScoped;

import java.time.ZoneId;
import java.time.ZonedDateTime;
import java.time.format.DateTimeFormatter;

import java.util.HashMap;
import java.util.Map;

import org.eclipse.microprofile.config.ConfigProvider;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.Optional;

/**
 * MQTT configuration as defined by Quarkus.
 * <p>
 * The data in this interface is taken from System properties, ENV variables,
 * .env file and more. Take a look at https://quarkus.io/guides/config-reference
 * to see how it works.
 */
class RestConfig {
    public static final String SEDA_MQTT_QUEUE_OUT = "seda:queue_out?multipleConsumers=true";

    public String storage_url;
    public String storage_topic;

    public Optional<String> storage_user;
    public Optional<String> storage_password;
}

/**
 * Route to read from REST.
 */
@ApplicationScoped
public class RestRoute extends RouteBuilder {
    private final RestConfig restConfig;

    public RestRoute()
    {
        this.restConfig = new RestConfig();

        this.restConfig.storage_url = ConfigProvider.getConfig().getValue("internal_mqtt.url", String.class);
        this.restConfig.storage_topic = ConfigProvider.getConfig().getValue("internal_mqtt.topic", String.class);
        this.restConfig.storage_user = ConfigProvider.getConfig().getOptionalValue("internal_mqtt.user", String.class);
        this.restConfig.storage_password = ConfigProvider.getConfig().getOptionalValue("internal_mqtt.password", String.class);
    } 

    @Override
    public void configure() {
        RestConfigLogger.log(restConfig);
        
        // Exposes REST connection
        // process and forward to SEDA_MQTT_QUEUE_OUT
        restConfiguration()
            //.component("netty-http")
            // .scheme("https")
            .apiContextPath("/api-doc")
            .apiProperty("api.title", "ODH inbound REST API")
            .apiProperty("api.version", "0.0.1")
            .bindingMode(RestBindingMode.auto)
            .contextPath("/")
            .host("0.0.0.0")
            .port(8080);
            

        from("rest:post:/{provider}")
            // Put body as string into header "rawdata" for later reuse
            .process(exchange -> {
                Map<String, Object> map = new HashMap<String, Object>();
                ObjectMapper objectMapper = new ObjectMapper();

                map.put("provider", exchange.getIn().getHeader("provider").toString());
                map.put("rawdata", exchange.getIn().getBody(String.class));
                map.put("timestamp", ZonedDateTime.now(ZoneId.of("UTC")).format(DateTimeFormatter.ISO_INSTANT));
                
                exchange.getMessage().setBody(objectMapper.writeValueAsString(map));
                exchange.getMessage().setHeader("provider", exchange.getIn().getHeader("provider").toString());
            })
            .log("REST| ${body}")
            .log("REST| ${headers}")
            .to(getInternalStorageQueueConnectionString())
            // .to(RestConfig.SEDA_MQTT_QUEUE_OUT)

            // reset and send response
            .setHeader(Exchange.HTTP_RESPONSE_CODE, constant(200))
            .setBody(constant(null))
            /*.endRest()*/;
    }

    private String getInternalStorageQueueConnectionString() {
        // TODO use AmazonSNS uri if needed
        // for testing purpose we use Mosquitto
        final StringBuilder uri = new StringBuilder(String.format("paho-mqtt5:%s?brokerUrl=%s&qos=2&retained=true", 
        restConfig.storage_topic, restConfig.storage_url));

        // Check if MQTT credentials are provided. If so, then add the credentials to the connection string
        restConfig.storage_user.ifPresent(user -> uri.append(String.format("&userName=%s", user)));
        restConfig.storage_password.ifPresent(pass -> uri.append(String.format("&password=%s", pass)));

        return uri.toString();
    }
}

/**
 * Utility class to log {@link MqttConfig}.
 */
final class RestConfigLogger {

    private static Logger LOG = LoggerFactory.getLogger(RestConfigLogger.class);

    private RestConfigLogger() {
        // Private constructor, don't allow new instances
    }

    /**
     * Log {@link MqttConfig}.
     *
     * @param config The {@link MqttConfig} to log.
     */
    public static void log(RestConfig config) {
    }
}