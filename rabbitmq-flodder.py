import pika

connection = pika.BlockingConnection(pika.ConnectionParameters('127.0.0.1'))
channel = connection.channel()

ROUTING_KEY = "skidata.carezza"
EXCHANGE_NAME = "skidata"

channel.exchange_declare(exchange=EXCHANGE_NAME, passive=True)
for i in range(100):
    channel.basic_publish(exchange=EXCHANGE_NAME, routing_key=ROUTING_KEY, body=("test %d" % i))