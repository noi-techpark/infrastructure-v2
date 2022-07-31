const { Timestamp } = require('mongodb');

async function monitorListingsUsingEventEmitter(client, mqttclient, topic, pipeline = [
    {"$match": {"operationType": "insert"}},
]) {
        let cursor = await getCursor(client);
        console.log(cursor);
        const checkpointInterrvall = 3600 * 1000; // 1 hour in milliseconds

        return new Promise((resolve) => {
        /**
         * difference between startAfter and startAtOperationTime
         * https://stackoverflow.com/questions/55184492/difference-between-resumeafter-and-startatoperationtime-in-mongodb-change-stream
        */
        // console.log(collection)
        const resumeToken = "8262CC86F3000000012B022C0100296E5A1004DAC285AFE7094F63923AF1BD7BBC196846645F6964006462CC86F2B8B4E7FCCD54B9F90004";
        const aftetTime = cursor ? cursor.timestamp : new Timestamp({ t: 0, i: 0 });
        console.log("Reading events starting from " + new Date(aftetTime.getHighBits() * 1000))
        console.log("Now is " + new Date())
        // const changeStream = collection.watch(pipeline, { startAfter : { _data: resumeToken} });
        // watching the whole deplyoment https://www.mongodb.com/docs/manual/changeStreams/
        const changeStream = client.watch(pipeline, { startAtOperationTime: aftetTime})
        // console.log(changeStream)

        let updating = false;
        changeStream.on('change', async (next) => {
            // console.log(changeStream.resumeToken);
            const operationTime = new Date(next.clusterTime.getHighBits() * 1000);
            if (cursor) {
                console.log(operationTime.getTime(), cursor.date.getTime());
                console.log(operationTime.getTime() - cursor.date.getTime(), checkpointInterrvall);
            }

            if (!updating && (!cursor || operationTime.getTime() - cursor.date.getTime() > checkpointInterrvall))
            {
                updating = true;
                cursor = await flushCheckpoint(client, cursor, next.clusterTime);
                updating = false;
                // console.log("new cursor", cursor);
            }
            mqttclient.publish(topic, JSON.stringify({
                id: next.documentKey._id.toString(),
                db: next.ns.db,
                collection: next.ns.coll
            }))
        });
    })

    //await closeChangeStream(60000, changeStream);
};

async function flushCheckpoint(client, old, timestamp) {
    const filter = old ? { _id: old._id } : {};
    const options = { upsert: true };
    const updateDoc = {
        $set: {
            timestamp: timestamp,
        },
    };
    const result = await client.db("admin").collection("notifier_checkpoint").updateOne(filter, updateDoc, options);
    // console.log(old, result);
    const date = new Date(timestamp.getHighBits() * 1000)
    return old ? { _id: old._id, timestamp: timestamp, date} : { _id: result.upsertedId, timestamp: timestamp, date };
}

async function getCursor(client) {
    const checkpoint = await client.db("admin").collection("notifier_checkpoint").findOne({ }, {});
    return checkpoint ? { ...checkpoint, date: new Date(checkpoint.timestamp.getHighBits() * 1000)} : null;
}

function closeChangeStream(timeInMs = 60000, changeStream) {
    return new Promise((resolve) => {
        setTimeout(() => {
            console.log("Closing the change stream");
            changeStream.close();
            resolve();
        }, timeInMs)
    })
};

module.exports = { monitorListingsUsingEventEmitter };