const { MongoClient } = require('mongodb');
const { monitorListingsUsingEventEmitter } = require('./changeStream');
const dotenv = require("dotenv")
const mqtt = require('mqtt');
const amqplib = require('amqplib');

/**
 * What are changestreams?: https://www.mongodb.com/docs/manual/changeStreams/
 * 
 * For the purpose of the PoC, we use a single MongoDB deployment as rawDataTable and we store data in {provider} db / {provider} collection
 * provider = flightdata -> data stored in flightdata/flightadata.
 * 
 * You might want to customize the logic about where data is stored. in that case you have to deploy a notifier for each MongoDB deployment.
 * 
 * We also use the same MongoDB deployment to store persistent data about the checkpoint used to restart processing data in the case of network or application failure.
 * checkpoint stored in admin/notifier_checkpoint.
 * 
 * You might want to store the checkpoint in a REDIS, filesystem, ...
 * In that case change the functions `flushCheckpoint` and `getCursor` according to your need
 */
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
    const client = new MongoClient(uri, { 
        useUnifiedTopology: true, 
        replicaSet: process.env.MONGODB_CONNECTION_REPLICA_SET 
    });

    /**
     * MQTT client for test env
     */

    // console.log(`connecting to mqtt ${process.env.NOTIFIER_QUEUE_URL}`)

    // const mqttclient  = mqtt.connect(process.env.NOTIFIER_QUEUE_URL)
    // const connect = new Promise(resolve => {
    //     mqttclient.on('connect', function () {
    //         console.log(`connected to mqtt ${process.env.NOTIFIER_QUEUE_URL}`)
    //         mqttclient.subscribe(process.env.NOTIFIER_QUEUE_TOPIC, function (err) {
    //             if (!err) {
    //                 resolve();
    //             } else {
    //                 throw new Error(`Could not connect to topic ${process.env.NOTIFIER_QUEUE_TOPIC}`)
    //             }
    //           })
            
    //     });
    // });

    let conn = undefined;

    while (!conn) {
        conn = await amqplib.connect(`amqp://${process.env.RABBITMQ_CLUSTER_URL}`).catch(err => console.log(err));
        await new Promise(r => setTimeout(r, 2000));
    }
    console.log(`connected to rabbitmq ${process.env.RABBITMQ_CLUSTER_URL}`);

    const readyExchange = "ready";
    const rabbitChannel = await conn.createChannel();
    rabbitChannel.assertExchange(readyExchange, 'direct', {
        durable: true
    });
    const fnPublish = (data) =>  {
        rabbitChannel.publish(readyExchange, "", Buffer.from(data));
    }
    
    // await connect;

    try {
        // Connect to the MongoDB cluster
        // ! TODO THe client / application should implement some kind of
        // ! checks to ensure the connection is alive, mongo is responsive and
        // ! the changestream is open and ready

        // ! the nofier may starts when the changestream is not ready
        // ! and it won't subscribe until a restart
        console.log(`connecting to db ${process.env.MONGODB_CONNECTION_STRING}`)
        await client.connect();

        // Make the appropriate DB calls
        await monitorListingsUsingEventEmitter(client, fnPublish, process.env.NOTIFIER_QUEUE_TOPIC);

    } finally {
        // Close the connection to the MongoDB cluster
        await client.close();
        console.log(`error connecting to db ${process.env.MONGODB_CONNECTION_STRING}`)
    }
}

main().catch(console.error);