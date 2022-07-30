// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-bean
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-seda
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-stream
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-paho-mqtt5
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-openapi-java

package it.bz.opendatahub.queue;

import org.apache.camel.builder.RouteBuilder;

import javax.enterprise.context.ApplicationScoped;

import java.util.Optional;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import org.eclipse.microprofile.config.ConfigProvider;

class QueueConfig {
    public static final String SEDA_MQTT_QUEUE_OUT = "seda:queue_out?multipleConsumers=true";

    public String url;
    public String topic;

    public Optional<String> user;
    public Optional<String> password;
}

/**
 * Store all MQTT messages.
 */
@ApplicationScoped
public class QueueRoute extends RouteBuilder {

    private QueueConfig queueConfig;

    public QueueRoute()
    {
        this.queueConfig = new QueueConfig();
        this.queueConfig.url = ConfigProvider.getConfig().getValue("queue_internal_storage.url", String.class);
        this.queueConfig.topic = ConfigProvider.getConfig().getValue("queue_internal_storage.topic", String.class);
        this.queueConfig.user = ConfigProvider.getConfig().getOptionalValue("queue_internal_storage.user", String.class);
        this.queueConfig.password = ConfigProvider.getConfig().getOptionalValue("queue_internal_storage.password", String.class);
    } 

    @Override
    public void configure() {
        QueueConfigLogger.log(queueConfig);

        // Use SEDA_MQTT_ALL_STREAM stream (all MQTT messages)
        // -> Write that data to table genericdata in batches
        from(QueueConfig.SEDA_MQTT_QUEUE_OUT)
                .routeId("[Route: to storage queue]")
                .log("QUEUE| ${body}")
                .log("QUEUE| ${headers}")
                .to(getInternalStorageQueueConnectionString());
    }

    private String getInternalStorageQueueConnectionString() {
        // TODO use AmazonSNS uri if needed
        // for testing purpose we use Mosquitto
        final StringBuilder uri = new StringBuilder(String.format("paho-mqtt5:%s?brokerUrl=%s", 
            queueConfig.topic, queueConfig.url));

        // Check if MQTT credentials are provided. If so, then add the credentials to the connection string
        queueConfig.user.ifPresent(user -> uri.append(String.format("&userName=%s", user)));
        queueConfig.password.ifPresent(pass -> uri.append(String.format("&password=%s", pass)));

        return uri.toString();
    }
}

/**
 * Utility class to log {@link QueueConfig}.
 */
final class QueueConfigLogger {

    private static Logger LOG = LoggerFactory.getLogger(QueueConfigLogger.class);

    private QueueConfigLogger() {
        // Private constructor, don't allow new instances
    }

    /**
     * Log {@link QueueConfig}.
     *
     * @param config The {@link QueueConfig} to log.
     */
    public static void log(QueueConfig config) {
        LOG.info("QUEUE_INTERNAL_STORAGE url: {}", config.url);
        LOG.info("QUEUE_INTERNAL_STORAGE topic: {}", config.topic);
        LOG.info("QUEUE_INTERNAL_STORAGE user: {}", config.user);
        LOG.info("QUEUE_INTERNAL_STORAGE password: {}", config.password);
    }
}