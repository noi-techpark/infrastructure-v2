// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-bean
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-openapi-java
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-paho
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-rabbitmq
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-rest
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-seda
// camel-k: dependency=mvn:org.apache.camel.quarkus:camel-quarkus-stream

package it.bz.opendatahub.inbound.rest;

import it.bz.opendatahub.RabbitMQConnection;
import it.bz.opendatahub.WrapperProcessor;

import org.apache.camel.Exchange;
import org.apache.camel.builder.RouteBuilder;
import org.apache.camel.model.rest.RestBindingMode;

import javax.enterprise.context.ApplicationScoped;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

/**
 * Route to read from REST.
 */
@ApplicationScoped
public class RestRoute extends RouteBuilder {
    private final RabbitMQConnection rabbitMQConfig;

    public RestRoute()
    {
        this.rabbitMQConfig = new RabbitMQConnection();
    } 

    @Override
    public void configure() {
        // Exposes REST connection
        // wrap message and send to RabbitMQ ingress
        restConfiguration()
            .apiContextPath("/api-doc")
            .apiProperty("api.title", "ODH inbound REST API")
            .apiProperty("api.version", "0.0.1")
            .bindingMode(RestBindingMode.auto)
            .contextPath("/")
            .host("0.0.0.0")
            .port(8080);
            

        from("rest:post:/*")
            // rest query params are passed as headers
            // therefore we have to delete reserved header names
            .removeHeaders("fastline")
            .removeHeaders("valid")

            .process(exchange -> WrapperProcessor.process(exchange, 
                exchange.getIn().getHeader(Exchange.HTTP_URI).toString()))
            // .log("REST| ${body}")
            // .log("REST| ${headers}")
            // we clear all CamelHttp since they could create problem
            // with other components
            .removeHeaders("CamelHttp*")
            .choice()
                // forward to fastline
                .when(header("fastline").isEqualTo(true))
                    // we handle the request as invalid and forward the encapsulated payload to 
                    // whatever mechanism we want to use to store malformed data
                    .to(this.rabbitMQConfig.getRabbitMQFastlineConnectionString())
            .end()
            .choice()
                // if the payload is not a valid json
            .when(header("valid").isEqualTo(false))
                // we handle the request as invalid and forward the encapsulated payload to 
                // whatever mechanism we want to use to store malformed data
                .to(this.rabbitMQConfig.getRabbitMQIngressDeadletterTo())
            .otherwise()
                // otherwise we forward the encapsulated message to the 
                // internal queue waiting to be written in rawDataTable
                .to(this.rabbitMQConfig.getRabbitMQIngressTo())
            .end()
            // reset and send responses
            .setBody(constant(null))
            .removeHeaders("*")
            .setHeader(Exchange.HTTP_RESPONSE_CODE, constant(200));
    }
}

final class RestConfigLogger {

    private static Logger LOG = LoggerFactory.getLogger(RestConfigLogger.class);

    private RestConfigLogger() {
        // Private constructor, don't allow new instances
    }
}