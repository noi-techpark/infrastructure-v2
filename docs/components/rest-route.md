# REST Route

![rest-route](../assets/mqtt-route.svg)

REST Route is the Camel Route responsible for listening to REST API calls on the perimeter.

## Provider

**REST Route** listens to any POST request on a single port (by default `8080`).

The path a provider uses to make requests is interpreted by the system as [Provider URI](../inbound.md#provider-uri) for the data transported in the request body.

For example, executing the following request:

```sh
 curl -H 'Content-Type: application/json' \
      -d '{ "title":"foo","body":"bar", "id": 1}' \
      -X POST \
      https://localhost:8080/skidata/carezza/paolina?fastline=true
```

will result in the system associating to the request body the following Provider URI

```
skidata/carezza/paolina?fastline=true
```

----

To make REST requests we suggest using [Insomnia](https://insomnia.rest/) or any other REST client.
