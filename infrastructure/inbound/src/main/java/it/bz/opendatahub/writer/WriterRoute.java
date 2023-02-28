// camel-k: dependency=mvn:io.quarkus:quarkus-mongodb-client
// camel-k: dependency=mvn:org.apache.camel:camel-jackson:3.6.0
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-bean
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-jackson
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-mongodb
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-paho
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-rabbitmq
// camel-k: dependency=mvn:org.apache.commons:commons-lang3:3.12.0


package it.bz.opendatahub.writer;

import org.apache.camel.builder.RouteBuilder;

import javax.enterprise.context.ApplicationScoped;
import java.util.Optional;

import java.util.HashMap;
import java.util.Map;
import java.util.Date;
import java.net.URI;
import java.net.URISyntaxException;
import java.time.*;

import org.apache.camel.component.jackson.JacksonDataFormat;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import org.eclipse.microprofile.config.ConfigProvider;

import java.time.ZoneId;
import java.time.ZonedDateTime;
import java.time.format.DateTimeFormatter;
import java.util.HashMap;

import org.apache.camel.Exchange;
import org.apache.camel.component.rabbitmq.RabbitMQConstants;
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

// import it.bz.opendatahub.RabbitMQConnection;

class MongoDBConnection {
    String host;
}

/**
 * Route to read from INTERNAL MQTT and store data in rawDataTable.
 */
@ApplicationScoped
public class WriterRoute extends RouteBuilder {

    private final RabbitMQConnection rabbitMQConfig;
    private final MongoDBConnection mongoDBConnection;

    public WriterRoute()
    {
        this.rabbitMQConfig = new RabbitMQConnection();
        this.mongoDBConnection = new MongoDBConnection();

        this.mongoDBConnection.host = ConfigProvider.getConfig().getValue("mongodb.host", String.class);
    } 

    @Override
    public void configure() {
        // Read from RabbitMQ ingress
        // Writes a valid BSON object to MongoDB
        // TODO Add throtling if needed
        // https://camel.apache.org/components/3.18.x/rabbitmq-component.html
        from(this.rabbitMQConfig.getRabbitMQIngressFrom())
            .routeId("[Route: Writer]")
            //.throttle(100).timePeriodMillis(10000)
            // .log("WRITE| ${body}")
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
                // we then compute the database connection using the message body (in this case we only care bout the field `provider`)
                // and store the connection string in the `database` header to be used later
                exchange.getMessage().setHeader("database", getDatabaseString((String)body.get("provider")));
            })
            // we don't use `.to()` because the connection string is dynamic and we use the previously set header `database`
            // to send the data to the database
            .recipientList(header("database"))
            .end();
    }

    /**
     * For the purpose of the PoC, we use a single MongoDB deployment as rawDataTable.
     * The db name is the first token of the provider uri
     * The collection name is the second token of the provider uri
     *      if there is only one token it will be used as collection as well
     * 
     * If we need to use multiple deployments or custom paths, you should edit this function.
     * References:
     * https://camel.apache.org/components/3.20.x/mongodb-component.html
     * https://quarkus.io/guides/mongodb
     */
    // ! invalid collection & db characters (on linux deployment): /\. "$ 
    // https://www.mongodb.com/docs/manual/reference/limits/#std-label-restrictions-on-db-names
    private String getDatabaseString(String provider) throws URISyntaxException {
        String[] tokens = new URI(provider).getPath().split("/");
        String db = tokens[0];
        String collection = tokens[0];
        if (tokens.length > 1) {
            collection = tokens[1];
        }
        final StringBuilder uri = new StringBuilder(String.format("mongodb:dummy?hosts=%s&database=%s&collection=%s&operation=insert", 
        this.mongoDBConnection.host, db, collection));
        return uri.toString();
    }
}

final class WriterConfigLogger {

    private static Logger LOG = LoggerFactory.getLogger(WriterConfigLogger.class);

    private WriterConfigLogger() {
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

    // Username is optional and may not be set
    Optional<String> user;

    // Password is optional and may not be set
    Optional<String> pass;
}

class RabbitMQConnection {

    static final String RABBITMQ_INGRESS_QUEUE = "ingress-q";
    static final String RABBITMQ_INGRESS_EXCHANGE = "ingress";
    static final String RABBITMQ_INGRESS_DEADLETTER_QUEUE = "ingress-dl-q";
    static final String RABBITMQ_INGRESS_DEADLETTER_EXCHANGE = "ingress-dl";
    static final String RABBITMQ_FASTLINE_EXCHANGE = "fastline";

    private static Logger LOG = LoggerFactory.getLogger(RabbitMQConnection.class);
    private RabbitMQConfig ingressConfig;

    public RabbitMQConnection() {
        this.ingressConfig = new RabbitMQConfig();
        this.ingressConfig.cluster = ConfigProvider.getConfig().getValue("rabbitmq.cluster", String.class);
        this.ingressConfig.user = ConfigProvider.getConfig().getOptionalValue("rabbitmq.user", String.class);
        this.ingressConfig.pass = ConfigProvider.getConfig().getOptionalValue("rabbitmq.pass", String.class);

        String user = this.ingressConfig.user.orElseGet(() -> "*** no user ***");
        String pass = this.ingressConfig.pass.map(p -> "*****").orElseGet(() -> "*** no password ***");

        LOG.info("RabbitMQ cluster: {}", this.ingressConfig.cluster);
        LOG.info("RabbitMQ user: {}", user);
        LOG.info("RabbitMQ password: {}", pass);
    }

    private String setAuth(StringBuilder uri) {
        // Check if RabbitMQ credentials are provided. If so, then add the credentials to the connection string
        this.ingressConfig.user.ifPresent(user -> uri.append(String.format("&username=%s", user)));
        this.ingressConfig.pass.ifPresent(pass -> uri.append(String.format("&password=%s", pass)));

        System.out.println(uri.toString());
        return uri.toString();
    }

    public String getRabbitMQIngressFrom() {
        final StringBuilder uri = new StringBuilder(String.format("rabbitmq:?"+
            "addresses=%s"+
            "&queue=%s"+
            "&autoAck=false"+
            // setting reQueue=true + autoAck=false messages not processed because of exceptions get requeued
            "&reQueue=true"+ 
            "&autoDelete=false"+
            "&skipExchangeDeclare=true"+
            "&skipQueueBind=true"+
            "&skipQueueDeclare=true",
            this.ingressConfig.cluster,
            RABBITMQ_INGRESS_QUEUE));

        return this.setAuth(uri);
    }

    public String getRabbitMQIngressTo() {
        final StringBuilder uri = new StringBuilder(String.format("rabbitmq:%s?"+
            "addresses=%s"+
            "&queue=%s"+
            "&autoDelete=false"+
            // https://stackoverflow.com/questions/14527185/activemq-i-cant-consume-a-message-sent-from-camel-using-inout-pattern
            // https://camel.apache.org/manual/exchange-pattern.html
            // we are using Event messages, therefore we have to specify the InOnly pattern
            // otherwise the component expects a reply
            "&exchangePattern=InOnly"+
            "&exchangeType=fanout",
            RABBITMQ_INGRESS_EXCHANGE,
            this.ingressConfig.cluster,
            RABBITMQ_INGRESS_QUEUE));

        return this.setAuth(uri);
    }

    public String getRabbitMQIngressDeadletterTo() {
        final StringBuilder uri = new StringBuilder(String.format("rabbitmq:%s?"+
            "addresses=%s"+
            "&queue=%s"+
            "&routingKey=ingress.*"+
            "&exchangeType=fanout"+
            "&exchangePattern=InOnly"+
            "&autoDelete=false", 
            RABBITMQ_INGRESS_DEADLETTER_EXCHANGE,
            this.ingressConfig.cluster,
            RABBITMQ_INGRESS_DEADLETTER_QUEUE));

        return this.setAuth(uri);
    }

    public String getRabbitMQFastlineConnectionString() {
        final StringBuilder uri = new StringBuilder(String.format("rabbitmq:%s?"+
            "addresses=%s"+
            "&passive=true"+
            "&exchangeType=topic"+
            "&skipQueueBind=true"+
            "&skipQueueDeclare=true"+
            "&exchangePattern=InOnly"+
            "&autoDelete=false"+
            "&declare=true", 
            RABBITMQ_FASTLINE_EXCHANGE, this.ingressConfig.cluster));

        return this.setAuth(uri);
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

        //https://github.com/Talend/apache-camel/blob/master/components/camel-rabbitmq/src/main/java/org/apache/camel/component/rabbitmq/RabbitMQConstants.java
        exchange.getMessage().setHeader(RabbitMQConstants.ROUTING_KEY, routeKey);
        //exchange.getMessage().setHeader(RabbitMQConstants.RABBITMQ_DEAD_LETTER_ROUTING_KEY, routeKey);

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
