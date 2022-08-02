// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-bean
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-paho-mqtt5
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-jackson
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-mongodb
// camel-k: dependency=mvn:io.quarkus:quarkus-mongodb-client
// camel-k: dependency=mvn:org.apache.camel:camel-jackson:3.6.0
// camel-k: dependency=mvn:org.apache.commons:commons-lang3:3.12.0


package it.bz.opendatahub.writer;

import org.apache.camel.builder.RouteBuilder;

import javax.enterprise.context.ApplicationScoped;

import java.util.HashMap;
import java.util.Map;
import java.util.Date;
import java.time.*;

import org.apache.commons.lang3.StringUtils;

import org.apache.camel.component.jackson.JacksonDataFormat;
import com.fasterxml.jackson.databind.ObjectMapper;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import org.eclipse.microprofile.config.ConfigProvider;

// import io.quarkus.mongodb.MongoClientName;

class WriterConfig {
    String url;
    String topic;
}

/**
 * Route to read from MQTT.
 */
@ApplicationScoped
public class WriterRoute extends RouteBuilder {
    private WriterConfig config;

    // @Inject
    // @MongoClientName("rawdata")
    // MongoClient mongoConnection;

    public WriterRoute()
    {
        this.config = new WriterConfig();
        this.config.url = ConfigProvider.getConfig().getValue("internal_mqtt.url", String.class);
        this.config.topic = ConfigProvider.getConfig().getValue("internal_mqtt.topic", String.class);
    } 

    @Override
    public void configure() {
        WriterConfigLogger.log(config);

        from(getInternalStorageQueueConnectionString())
            .end()
            .log("${body}")
            .unmarshal(new JacksonDataFormat())
            .process(exchange -> {
                Map<String, Object> body = (HashMap<String, Object>)exchange.getMessage().getBody(Map.class);
                Object timestamp = body.get("timestamp");
                if (timestamp != null)
                {
                    Instant instant = Instant.parse((String)timestamp);
                    Date dateTimestamp = Date.from(instant);
                    body.put("bsontimestamp", dateTimestamp);
                }
                exchange.getMessage().setBody(body);
                exchange.getMessage().setHeader("database", getDatabaseString((String)body.get("provider")));
            })
            .log("${body[provider]}")
            // .log("${body.GetProvider}")
            // .log("${body.GetRawdata}")
            // .to("mongodb://camelMongoClient?database=test&collection=test&operation=save");
            .recipientList(header("database"));
            //?hosts=localhost:30001
    }

    private String getDatabaseString(String provider) {
        String cleanProvider = StringUtils.stripStart(provider, "/");
        final StringBuilder uri = new StringBuilder(String.format("mongodb://camelMongoClient?database=%s&collection=%s&operation=insert", 
        cleanProvider, cleanProvider));

        // // Check if MQTT credentials are provided. If so, then add the credentials to the connection string
        // config.user().ifPresent(user -> uri.append(String.format("&userName=%s", user)));
        // config.password().ifPresent(pass -> uri.append(String.format("&password=%s", pass)));
        System.out.println(uri.toString());

        return uri.toString();
    }

    private String getInternalStorageQueueConnectionString() {
        // TODO use AmazonSNS uri if needed
        // for testing purpose we use Mosquitto
        final StringBuilder uri = new StringBuilder(String.format("paho-mqtt5:%s?brokerUrl=%s&qos=2", 
            config.topic, config.url));
        return uri.toString();
    }
}

/**
 * Utility class to log {@link WriterConfig}.
 */
final class WriterConfigLogger {

    private static Logger LOG = LoggerFactory.getLogger(WriterConfigLogger.class);

    private WriterConfigLogger() {
        // Private constructor, don't allow new instances
    }

    /**
     * Log {@link WriterConfig}.
     *
     * @param config The {@link WriterConfig} to log.
     */
    public static void log(WriterConfig config) {
        LOG.info("WRITER|INTERNAL_MQTT url: {}", config.url);
        LOG.info("WRITER|INTERNAL_MQTT topic: {}", config.topic);
    }
}
