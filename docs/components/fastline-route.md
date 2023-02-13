# Fastline Route

![fastline-route](../assets/fastline-route.svg)

**Fastline Route** is an example of the flexibility of the new architecture.

Some provider could request their data is both transformed and served using the standard flow and served as-is in real-time.

By specifying the `fastline=true` **query parameter** of the [Provider URI](../inbound.md#provider-uri), the inbound APIs will directly transfer the data to a **WebSocket** outbound API. Clients can connect to the WebSocket and receive real-time as-is data.

In parallel the data will also follow the standard flow, being written in the [Raw Data Table](../raw-data-table.md), routed, transformed and served.

## Example

For the sake of simplicity, we implemented a single WebSocket which broadcasts all data sent to a specific RabbitMQ's Queue (`fastline-q`) to any subscribed client using [Camel Jetty WebSocket Component](https://camel.apache.org/components/3.20.x/websocket-component.html).

In the future, the system will expose multiple WebSocket, one for each provider, and route the fastline message using a complementary [Router Route](router-route.md).

We suggest [Firecamp](https://chrome.google.com/webstore/detail/firecamp-a-multi-protocol/eajaahbjpnhghjcdaclbkeamlkepinbl) Chrome Extension to subscribe to the WebSocket and verify that real-time messages are correctly delivered.