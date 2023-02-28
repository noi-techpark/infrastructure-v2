// camel-k: dependency=mvn:io.quarkus:quarkus-mongodb-client
// camel-k: dependency=mvn:org.apache.camel:camel-jackson:3.6.0
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-bean
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-jackson
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-mongodb
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-paho
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-rabbitmq
// camel-k: dependency=mvn:org.apache.commons:commons-lang3:3.12.0

package it.bz.opendatahub.pull;

import org.apache.camel.builder.RouteBuilder;
import org.apache.camel.AsyncCallback;
import org.apache.camel.AsyncProcessor;
import org.apache.camel.Exchange;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import javax.enterprise.context.ApplicationScoped;

import java.util.HashMap;
import java.util.Map;
import java.util.Optional;
import java.util.concurrent.CompletableFuture;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;

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
// import it.bz.opendatahub.WrapperProcessor;

class PullAsyncProcessor implements AsyncProcessor {
    private HttpClient client;

    private String[] endpoints;
    private String[] endpointKeys;

    class Calls {
        private HttpClient client;

        private Integer resolvedEndpoints;
        private Map<String, String> callToKey;
        private Integer endpointCount;
        private HashMap<String, Object> data;

        public Calls(HttpClient client) {
            this.client = client;
            this.callToKey = new HashMap<>();
            this.data = new HashMap<>();
            this.resolvedEndpoints = 0;
        }

        public void call(final Exchange exchange, AsyncCallback asyncCallback) {
            this.endpointCount = endpoints.length;

            for (int i = 0; i < this.endpointCount; i++) {
                if (this.endpointCount > 1) {
                    this.callToKey.put(endpoints[i], endpointKeys[i]);
                    System.out.println("setting: " + endpoints[i] + ", " + endpointKeys[i]);
                }
                System.out.println("calling endpoint: " + endpoints[i]);
                this.client
                        .sendAsync(
                                HttpRequest.newBuilder()
                                        .GET()
                                        .uri(URI.create(endpoints[i]))
                                        // .header("Accept", "application/json")
                                        .build(),
                                HttpResponse.BodyHandlers.ofString())
                        // .thenApply(HttpResponse::body)
                        .thenApply(data -> {
                            System.out.println("status code: " + data.statusCode());
                            // callback
                            if (data.statusCode() != 200) {
                                exchange.getMessage().setHeader("failed", true);
                            }

                            if (this.endpointCount == 1) {
                                exchange.getMessage().setBody(data.body());
                            } else {
                                this.data.put(this.callToKey.get(data.request().uri().toString()), data.body());

                            }

                            this.resolvedEndpoints++;
                            if (this.resolvedEndpoints == this.endpointCount) {
                                if (this.endpointCount > 1) {
                                    ObjectMapper objectMapper = new ObjectMapper();

                                    try {
                                        exchange.getMessage().setBody(objectMapper.writeValueAsString(this.data));
                                    } catch (JsonProcessingException e) {
                                        // TODO Handle exception by setting the key as unretriveable
                                        // TODO and setting exchange header as failed
                                        System.out.println("could not marshal body: " + e.getMessage());
                                        exchange.getMessage().setHeader("failed", true);
                                    }
                                }
                                asyncCallback.done(true);
                            }
                            return data;
                        });

            }
        }
    }

    public PullAsyncProcessor(HttpClient client) {
        this.client = client;
        this.endpoints = ConfigProvider.getConfig().getValue("pull.endpoints", String.class).split(",");

        Optional<String> endpointKeysString = ConfigProvider.getConfig().getOptionalValue("pull.endpointKeys",
                String.class);

        // TODO check there are keys if this.endpointCount > 1
        if (endpointKeysString.isPresent()) {
            this.endpointKeys = endpointKeysString.get().split(",");
            // TODO if keys, check this.endpointKeys length == this.endpointCount
        }
    }

    @Override
    public boolean process(final Exchange exchange, AsyncCallback asyncCallback) {
        Calls calls = new Calls(this.client);
        calls.call(exchange, asyncCallback);
        return true;
    }

    @Override
    public void process(Exchange exchange) throws Exception {
        throw new IllegalStateException("Should never be called");
    }

    @Override
    public CompletableFuture<Exchange> processAsync(Exchange exchange) {
        throw new IllegalStateException("Should never be called");
    }
}

@ApplicationScoped
public class PullRoute extends RouteBuilder {
    private final RabbitMQConnection rabbitMQConfig;
    private HttpClient client;

    private String provider;

    public PullRoute() {
        this.rabbitMQConfig = new RabbitMQConnection();
        client = HttpClient.newBuilder()
                .version(HttpClient.Version.HTTP_2)
                .build();

        this.provider = ConfigProvider.getConfig().getValue("pull.provider", String.class);
        System.out.println("starting pull route for provider: " +  this.provider);
    }

    @Override
    public void configure() {
        from("cron:tab?schedule=0+*+*+*+*+?")
                .routeId("[Route: Pull]" + this.provider)
                .setBody(simple("${null}"))
                .log("received")
                .process(new PullAsyncProcessor(this.client))
                .process(exchange -> WrapperProcessor.process(exchange, this.provider))
                .log("done")
                // .to("file:bar?doneFileName=done")
                .choice()
                    // forward to fastline
                    .when(header("fastline").isEqualTo(true))
                        // we handle the request as invalid and forward the encapsulated payload to 
                        // whatever mechanism we want to use to store malformed data
                        .to(this.rabbitMQConfig.getRabbitMQFastlineConnectionString())
                .end()
                .choice()
                    // if the payload is not a valid json
                    .when(header("failed").isEqualTo(true))
                        // we handle the request as invalid and forward the encapsulated payload to
                        // whatever mechanism we want to use to store malformed data
                        .to(this.rabbitMQConfig.getRabbitMQIngressDeadletterTo())
                    .otherwise()
                        // otherwise we forward the encapsulated message to the
                        // internal queue waiting to be written in rawDataTable
                        .to(this.rabbitMQConfig.getRabbitMQIngressTo())
                .end();
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
