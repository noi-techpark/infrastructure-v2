# MongoDB Raw Data Table

| From | To | Note |
| - | - | - |
| [WriterRoute](./components/writer-route.md) | [Notifier](./components/notifier.md) |  |

The **Raw Data Table** is a MongoDB cluster.

We choose MongoDB because it allows the system to insert non-structured data, it provides **replication** and **sharding** and also a mechanism to listen to any data change called `change stream`.

## Replica Set
To enable the `change stream`, the essential mechanism thanks to the [notifier](./components/notifier.md) is able to listen to any change in the database and promptly inform the right [transformer](./components/transformer.md), we need to deploy a MongoDB cluster in **replicaSet Mode**.

Read the [manual](https://www.mongodb.com/docs/manual/tutorial/deploy-replica-set/) about replica sets.

## Connect to a replica set

To connect to the MongoDB deployment we suggest using [Compass](https://www.mongodb.com/products/compass). Be aware that being the deployment a `Replica Set`, the URI string must be properly configured ([Doc](https://www.mongodb.com/docs/manual/reference/connection-string/)) and you have to check **Direct Connection** in the **Advanced Connection Options** of Compass.