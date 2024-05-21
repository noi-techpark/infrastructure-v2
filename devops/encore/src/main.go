package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strconv"
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
	From       time.Time
	To         time.Time
}

func parseTime(str string) time.Time {
	i, err := strconv.Atoi(str)
	failOnError(err, "Error parsing time "+str)
	return time.UnixMilli(int64(i))
}

func conf() {
	Config.Rabbit = os.Getenv("RABBIT_URI")
	Config.Mongo = os.Getenv("MONGO_URI")
	Config.Db = os.Getenv("DB")
	Config.Collection = os.Getenv("COLLECTION")
	Config.Queue = os.Getenv("QUEUE")
	Config.From = parseTime(os.Getenv("FROM"))
	Config.To = parseTime(os.Getenv("TO"))

	c, err := json.MarshalIndent(Config, "", " ")
	failOnError(err, "Unable to dump config")
	log.Print("Config: %s", string(c))
}

func main() {
	conf()

	msgs := make(chan M)

	go func() {
		cl, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(Config.Mongo))
		failOnError(err, "Mongo fail")

		defer func() {
			failOnError(cl.Disconnect(context.TODO()), "Disconnect fail")
		}()

		col := cl.Database(Config.Db).Collection(Config.Collection)
		filter := bson.D{
			{Key: "bsontimestamp", Value: bson.D{{Key: "$gte", Value: primitive.NewDateTimeFromTime(Config.From)}}},
			{Key: "bsontimestamp", Value: bson.D{{Key: "$lt", Value: primitive.NewDateTimeFromTime(Config.To)}}},
		}

		opts := options.Find().SetSort(bson.D{{Key: "bsontimestamp", Value: 1}})
		cur, err := col.Find(context.TODO(), filter, opts)
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
	for m := range msgs {
		j, err := json.Marshal(m)
		failOnError(err, "Json fail")
		log.Print("pushing message")
		failOnError(ch.PublishWithContext(context.TODO(), amqp.DefaultExchange, q.Name, false, false, amqp.Publishing{ContentType: "application/json", Body: j}), "Push fail")
	}

	log.Println("Job done!")
}
