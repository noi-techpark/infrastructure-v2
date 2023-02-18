# Wrapper Processor

The **Wrapper Procesor** is a custom `camel processor` implementation shared by all inbound routes (MQTT, REST and Pull). By "processor" we mean it's a portion of code run whenever some data enter any of the inbound routes. This code is therefore an extension of those routes.

The only purpose of this component is to wrap the raw data ingested in the system in a standard "envelope".

For the purpose of the PoC, we decided to wrap the data in a simple structure defined as follows:

```json
{
    "timestamp": "the timestamp of when the message has entered the system",
    "provider": "the provider URI",
    "rawdata": "an escaped payload containing the raw data ingested"
}
```

To know more about **Provider URI** read its [associated documentation](../inbound.md#provider-uri).

The above message will be then written as it is in the [Raw Data Table](../raw-data-table) by the [Writer Route](./writer-route.md).


The logic of the **Wrapper Processor** can be modified to include more or fewer fields, depending on the further development of the architecture.

We suggest keeping the logic **as simple as possible** as its priority is to persist the data in the **Raw Data Table** as soon as possible minimizing the risks of failures.