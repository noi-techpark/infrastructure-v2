// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-bean
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-paho
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
 * Route to read from INTERNAL MQTT and store data in rawDataTable.
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

        // Read from Internal MQTT
        // Writes a valid BSON object to MongoDB
        // TODO Add throtling if needed
        // TODO If error occurs, don't ACK message
        from(getInternalStorageQueueConnectionString())
            .end()
            .log("${body}")
            .unmarshal(new JacksonDataFormat())
            .process(exchange -> {
                // First we unmarshal the payload
                Map<String, Object> body = (HashMap<String, Object>)exchange.getMessage().getBody(Map.class);
                Object timestamp = body.get("timestamp");
                // we convert the timestamp field into a valid BSON TimeStamp
                if (timestamp != null)
                {
                    Instant instant = Instant.parse((String)timestamp);
                    Date dateTimestamp = Date.from(instant);
                    body.put("bsontimestamp", dateTimestamp);
                }
                exchange.getMessage().setBody(body);
                // we then compute the database connection usong the message body (in this case we only care bout the field `provider`)
                // and store the connection string in the `database` header to be used later
                //exchange.getMessage().setHeader("database", getDatabaseString((String)body.get("provider")));
            })
            // we don't use `.to()` because the connection string is dynamic and we use the previously set header `database`
            // to send the data to the database
            .recipientList(header("database"));
    }

    /**
     * For the purpose of the PoC, we use a single MongoDB deployment as rawDataTable and we store data in {provider} db / {provider} collection
     * provider = flightdata -> data stored in flightdata/flightadata.
     * 
     * If we need to use multiple deployments or custom paths, you should edit this function.
     * References:
     * https://camel.apache.org/camel-quarkus/2.10.x/reference/extensions/mongodb.html
     * https://quarkus.io/guides/mongodb
     */
    private String getDatabaseString(String provider) {
        String cleanProvider = StringUtils.stripStart(provider, "/");
        final StringBuilder uri = new StringBuilder(String.format("mongodb://camelMongoClient?database=%s&collection=%s&operation=insert", 
        cleanProvider, cleanProvider));

        return uri.toString();
    }

    // When using Mosquitto
    //      Connecting to the internal MQTT is not trivial and both publishers and subscribers follow a certain
    //      agreement to ensure no message will be lost:
    //      Publishers MUST publish with QoS >= 1
    //      Subscribers MUST connect with QoS >= 1
    //      Subscribers MUST connect with cleanStart = false
    //      ALL Subscribers in ALL pods must connect with a unique clientId which can't change at pod restart
    //      Read https://www.hivemq.com/blog/mqtt-essentials-part-7-persistent-session-queuing-messages/
    //      https://stackoverflow.com/questions/52439954/get-all-messages-after-the-client-has-re-connected-to-the-mqtt-broker
    //      to know more aboutn Persistent COnnection 
    private String getInternalStorageQueueConnectionString() {
        // TODO use AmazonSNS uri if needed
        // paho-mqtt5 has some problem with the retained messages on startup
        // -> https://stackoverflow.com/questions/56248757/camel-paho-routes-not-receiving-offline-messages-while-connecting-back

        // therefore we use paho
        final StringBuilder uri = new StringBuilder(String.format("paho:%s?brokerUrl=%s&cleanSession=false&qos=2&clientId=writer-route", 
            config.topic, config.url));
        return uri.toString();
    }
}

final class WriterConfigLogger {

    private static Logger LOG = LoggerFactory.getLogger(WriterConfigLogger.class);

    private WriterConfigLogger() {
        // Private constructor, don't allow new instances
    }

    public static void log(WriterConfig config) {
        LOG.info("WRITER|INTERNAL_MQTT url: {}", config.url);
        LOG.info("WRITER|INTERNAL_MQTT topic: {}", config.topic);
    }
}
