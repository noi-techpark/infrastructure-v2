# Wrapper Processor

The **Wrapper Procesor** is a custom `camel processor` implementation shared by all inbound routes (MQTT, REST and Pull). By "processor" wr mean it's a portion of code run whenever some data enter any of the inbound route. This code is therefore an extension of those routes.

It's soley purpose is to wrap the raw data ingested in the system in a standard "envelope".

For the purpose of the PoC, we decided to wrap the data in a simple structure so defined:

```json
{
    "timestamp": "timestamp when the message has entered the system",
    "provider": "provider URI",
    "rawdata": "string containing the escaped data ingested"
}
```

To know more about **Provider URI** read the [doc page](../inbound.md#provider-uri).

The above mesage will be then written as it is in the [Raw Data Table](../raw-data-table) by the [Writer Route](write-route).


The logic of the **Wrapper Processor** can be modified to include more or less fields, depending of the further development.

We suggest to keep the logic **as simple as possible**, since the proprity it to persist the data in the **Raw Data Table** as soon as possible minimizing the risks of failures.