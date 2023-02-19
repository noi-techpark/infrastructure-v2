package it.bz.opendatahub.transformer; 

import com.rabbitmq.client.Channel;
import com.rabbitmq.client.Connection;
import com.rabbitmq.client.Consumer;
import com.rabbitmq.client.ConnectionFactory;
import com.rabbitmq.client.AMQP.Queue.DeclareOk;
import com.rabbitmq.client.BuiltinExchangeType;
import java.io.IOException;

import io.quarkus.runtime.QuarkusApplication;
import io.quarkus.runtime.Quarkus;

// ack timeout https://www.rabbitmq.com/consumers.html#acknowledgement-timeout
// RabbitMQ package doc: https://rabbitmq.github.io/rabbitmq-java-client/api/current/index.html
public class Poller implements QuarkusApplication {

  private static final String EXCHANGE_NAME = "routed";
  private static final String QUEUE_NAME = "suedtirol.wein2";
  private static final String ROUTING_KEY = "suedtirol.wein2";

  private static final String POSTHOOK_EXCHANGE = "push-update";
  private static final String POSTHOOK_ROUTING_KEY = "suedtirol.wein2";

  @Override
  public int run(String... args) throws Exception {
      ConnectionFactory factory = new ConnectionFactory();
      factory.setAutomaticRecoveryEnabled(true);
      factory.setUsername("guest");
      factory.setPassword("guest");
      factory.setHost("rabbitmq-outbound");
      factory.setPort(5672);

      Connection connection = factory.newConnection();
      Channel channel = connection.createChannel();
      // max chunk size
      channel.basicQos(10, true);
      Channel postHookChannel = connection.createChannel();
  
      channel.exchangeDeclarePassive(EXCHANGE_NAME);
      DeclareOk ok =  channel.queueDeclare(QUEUE_NAME, true, false, false, null);
      channel.queueBind(ok.getQueue(), EXCHANGE_NAME, ROUTING_KEY);

      postHookChannel.exchangeDeclare(POSTHOOK_EXCHANGE, BuiltinExchangeType.TOPIC, true, false, false, null);
  
      System.out.println(" [*] Waiting for messages. To exit press CTRL+C");

      ConsumerImpl consumer = new ConsumerImpl(ok.getQueue(), channel);
      consumer.setPostHook(postHookChannel, POSTHOOK_EXCHANGE, POSTHOOK_ROUTING_KEY);
      consumer.setBatch(10, 40000 /*40sec*/);

      channel.basicConsume(ok.getQueue(), false, consumer);

      Quarkus.waitForExit();
      return 0;
  }
}