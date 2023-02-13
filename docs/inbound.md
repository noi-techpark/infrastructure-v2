# Inbounds

In this page the specifications for the data pushed/pulled by the **Inbound Apis** will be explained.

The concept illustrated applies both to data pushed by Providers onto Open Data Hub and to data pulled by Open Data Hub from providers regardless the protocol used (MQTT, REST, ...).

## Message wrapping
Any data ingested thought the inbounds is incapsulated in a standardized message using [WrapperProcessor](componenets/wrapper-processor) and sent To RabbitMQ.

## Provider URI

When a provider pushes data to Open Data Hub's inbounds, or when Hopen Data Hub collects data from a provider, the system needs to identify where the data comes from.

We designed the system to work with `Provider URI`s which identify who sent/from who we collect the data.

The `Provider URI` is specified in inbound and it's propagated to the whole system.

`Provider URI` has the same format of a standard **Web URI**

Example:

```
skidata/carezza/paolina?fastline=true
```

The `path` of the `URI` is used by the system to identify `provider` and eventual `subdomains`.
There is no limit to the amount of `subdomains` and it could be omitted:

```
skidata?fastline=true
```

Later the system will use `provider` and `subdomains` to [write](./write-route.md) in the **Raw Data Table** and how to [route](./router-route.md) the message to the right **Transformer**.

The `query parameters` of the `URI` are used by the system to perform special operation or routing of the data ingested.

For example the above `Provider URI` has the query parameter 
```
fastline=true
```

MQTT Route will capture this parameter and detect the request by the provider to use the [Fastline Outbound](./fastline-route.md) in order to expose the raw content of the message in real-time.

## Dead Letters

Malformed `Provider URIs` , message which do not pass the validation specified by Open Data Hub's policies or *data collection failure* are routed to a **Dead Letter exchange** in [RabbitMQ](./rabbitmq.md) which collects all messaged ingested in the inbounds which are not compliant to Open Data Hub standards.
