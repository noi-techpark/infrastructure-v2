package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Print(err)
		log.Panic(msg)
	}
}

type M struct {
	Id         string `json:"id"`
	Db         string `json:"db"`
	Collection string `json:"collection"`
}

type D struct {
	Id primitive.ObjectID `bson:"_id"`
	Ts primitive.DateTime `bson:"bsontimestamp"`
}

var Config struct {
	Rabbit     string
	Mongo      string
	Db         string
	Collection string
	Queue      string
	Query      string
}

func conf() {
	Config.Rabbit = os.Getenv("RABBIT_URI")
	Config.Mongo = os.Getenv("MONGO_URI")
	Config.Db = os.Getenv("DB")
	Config.Collection = os.Getenv("COLLECTION")
	Config.Queue = os.Getenv("QUEUE")
	Config.Query = os.Getenv("QUERY")

	c, err := json.MarshalIndent(Config, "", " ")
	failOnError(err, "Unable to dump config")
	log.Print("Config:", string(c))
}

const batchSize = 1000

func main() {
	conf()

	msgs := make(chan M, batchSize*2)

	go func() {
		cl, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(Config.Mongo))
		failOnError(err, "Mongo fail")

		defer func() {
			failOnError(cl.Disconnect(context.TODO()), "Disconnect fail")
		}()

		col := cl.Database(Config.Db).Collection(Config.Collection)

		opts := options.Find().SetSort(bson.D{{Key: "bsontimestamp", Value: 1}}).SetBatchSize(batchSize)

		var queryDoc interface{}
		failOnError(bson.UnmarshalExtJSON([]byte(Config.Query), false, &queryDoc), "Failed to unmarshal query")

		cur, err := col.Find(context.TODO(), queryDoc, opts)
		failOnError(err, "Cursor fail")

		for cur.Next(context.TODO()) {
			var res D
			failOnError(cur.Decode(&res), "Decode fail")
			msgs <- M{
				Id:         res.Id.String(),
				Db:         col.Database().Name(),
				Collection: col.Name(),
			}
		}
		close(msgs)
	}()

	con, err := amqp.Dial(Config.Rabbit)
	failOnError(err, "Connection fail")
	defer con.Close()

	ch, err := con.Channel()
	failOnError(err, "Channel fail")
	defer ch.Close()

	q, err := ch.QueueDeclare(Config.Queue,
		true,
		false,
		false,
		false,
		nil)

	failOnError(err, "Q fail")

	count := 0

	t := time.NewTicker(5 * time.Second)
	go func() {
		for range t.C {
			if count > 0 {
				log.Printf("...Pushed %d records", count)
			}
		}
	}()
	for m := range msgs {
		j, err := json.Marshal(m)
		failOnError(err, "Json fail")
		failOnError(ch.PublishWithContext(context.TODO(), amqp.DefaultExchange, q.Name, false, false, amqp.Publishing{ContentType: "application/json", Body: j}), "Push fail")
		count++
	}

	t.Stop()
	log.Printf("Pushed %d records", count)
	log.Println("Job done!")
}
