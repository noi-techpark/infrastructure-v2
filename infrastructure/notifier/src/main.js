const { MongoClient } = require('mongodb');
const { monitorListingsUsingEventEmitter } = require('./changeStream');
const dotenv = require("dotenv")
const mqtt = require('mqtt');

async function main() {
    dotenv.config()

    /**
     * Connection URI. Update <username>, <password>, and <your-cluster-url> to reflect your cluster.
     * See https://docs.mongodb.com/drivers/node/ for more details
     */
    /**
     * Connectcion uri doc: https://www.mongodb.com/docs/manual/reference/connection-string/
    */
    const uri = process.env.MONGODB_CONNECTION_STRING;
    
    /**
     * The Mongo Client you will use to interact with your database
     * See https://mongodb.github.io/node-mongodb-native/3.6/api/MongoClient.html for more details
     * In case: '[MONGODB DRIVER] Warning: Current Server Discovery and Monitoring engine is deprecated...'
     * pass option { useUnifiedTopology: true } to the MongoClient constructor.
     * const client =  new MongoClient(uri, {useUnifiedTopology: true})
     */
    const client = new MongoClient(uri, { useUnifiedTopology: true, replicaSet: process.env.MONGODB_CONNECTION_REPLICA_SET });

    /**
     * MQTT client for test env
     */

     console.log(`connecting to mqtt ${process.env.NOTIFIER_QUEUE_URL}`)

    const mqttclient  = mqtt.connect(process.env.NOTIFIER_QUEUE_URL)
    const connect = new Promise(resolve => {
        mqttclient.on('connect', function () {
            console.log(`connected to mqtt ${process.env.NOTIFIER_QUEUE_URL}`)
            mqttclient.subscribe(process.env.NOTIFIER_QUEUE_TOPIC, function (err) {
                if (!err) {
                    resolve();
                } else {
                    throw new Error(`Could not connect to topic ${process.env.NOTIFIER_QUEUE_TOPIC}`)
                }
              })
            
        });
    });
    
    await connect;

    try {
        // Connect to the MongoDB cluster
        console.log(`connecting to db ${process.env.MONGODB_CONNECTION_STRING}`)
        await client.connect();

        // Make the appropriate DB calls
        await monitorListingsUsingEventEmitter(client, mqttclient, process.env.NOTIFIER_QUEUE_TOPIC);

    } finally {
        // Close the connection to the MongoDB cluster
        await client.close();
        console.log(`error connecting to db ${process.env.MONGODB_CONNECTION_STRING}`)
    }
}

main().catch(console.error);