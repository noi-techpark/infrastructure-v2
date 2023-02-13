# Writer Route

![writer route](../assets/writer-route.svg)

**Writer Route** has the task to pull messages from [RabbitMQ](../rabbitmq), more precisely from the **queue** bounded to the `ingress` exchange using `ingress.#` as [Routing Key](../rabbitmq.md#routing-key), and write them in the [Raw Data Table](../raw-data-table.md).

To do so it uses [Camel RabbitMQ Component](https://camel.apache.org/components/3.20.x/spring-rabbitmq-component.html) to read, and [Camel MongoDB Component](https://camel.apache.org/components/3.20.x/mongodb-component.html) to write.

## Database and Collection

One key feature of **Writer Route** is to chose the right `database` and `collection` where to store the data, give an [Provider URI](../inbound.md#provider-uri).

To do that, it uses the **first** segment of Provider URIs **path** as `database`, and the second as `collection` if present.

If the Provider URI has only one segment in the **path**, if will used for the `collection` as well.

| Provider URI | Database | Collection |
| - | - | - |
| skidata/ortisei/easy?fastline=true | skidata | ortisei |
| skidata/ortisei | skidata | ortisei |
| skidata?fastline=true | skidata | skidata |