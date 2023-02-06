// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-bean
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-paho
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-jackson
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-mongodb
// camel-k: dependency=mvn:io.quarkus:quarkus-mongodb-client
// camel-k: dependency=mvn:org.apache.camel:camel-jackson:3.6.0

package it.bz.opendatahub.pull;

import org.apache.camel.builder.RouteBuilder;
import org.apache.camel.AsyncCallback;
import org.apache.camel.AsyncProcessor;
import org.apache.camel.Exchange;

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

import it.bz.opendatahub.RabbitMQConnection;
import it.bz.opendatahub.WrapperProcessor;

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
        from("cron:tab?schedule=0/10+*+*+*+*+?")
                .routeId("[Route: Pull]" + this.provider)
                .setBody(simple("${null}"))
                .log("received")
                .process(new PullAsyncProcessor(this.client))
                .process(exchange -> WrapperProcessor.process(exchange, this.provider))
                .log("done")
                // .to("file:bar?doneFileName=done")
                .choice()
                // if the payload is not a valid json
                .when(header("failed").isEqualTo(true))
                // we handle the request as invalid and forward the encapsulated payload to
                // whatever mechanism we want to use to store malformed data
                .to(this.rabbitMQConfig.getRabbitMQIngressDeadletterConnectionString())
                .otherwise()
                // otherwise we forward the encapsulated message to the
                // internal queue waiting to be written in rawDataTable
                .to(this.rabbitMQConfig.getRabbitMQIngressConnectionString())
                .end()
                // .to(this.rabbitMQConfig.getRabbitMQIngressConnectionString())
                .end();
    }
}