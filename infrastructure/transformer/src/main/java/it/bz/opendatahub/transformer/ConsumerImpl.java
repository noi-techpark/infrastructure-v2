// SPDX-FileCopyrightText: NOI Techpark <digital@noi.bz.it>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package it.bz.opendatahub.transformer; 

import com.rabbitmq.client.Channel;
import com.rabbitmq.client.DefaultConsumer;
import com.rabbitmq.client.MessageProperties;
import com.rabbitmq.client.Envelope;
import com.rabbitmq.client.ShutdownSignalException;
import com.rabbitmq.client.AMQP.BasicProperties;
import java.io.IOException;
import java.util.HashMap;
import java.util.ArrayList;
import java.util.TimerTask;
import java.util.Timer;
import java.util.Date;
import java.util.Optional;

class BatchTimeout extends TimerTask {

  private ConsumerImpl consumer;
  private boolean scheduled = false;

  BatchTimeout(ConsumerImpl consumer) {
    this.consumer = consumer;
  }

  @Override
  public void run() {
    this.scheduled = false;
    try {
      this.consumer.flushBatch();
    } catch (IOException ex) {
      System.out.println("could not flush batch: " + ex.getMessage());
    }
  }

  public void schedule() {
    this.scheduled = true;
  }

  public boolean hasBeenScheduled() {
      return this.scheduled;
  }
}

class Message
{
  public Envelope envelope;
  public BasicProperties properties;
  public byte[] body;
  public boolean acknowledgable;

  Message (Envelope envelope, BasicProperties properties, byte[] body) {
    this.envelope = envelope;
    this.properties = properties;
    this.body = body;
    this.acknowledgable = false;
  }
}

public class ConsumerImpl extends DefaultConsumer {
  private String queue;
  private Integer batchSize;
  private Integer batchTimeout;
  private String postHookExchange;
  private String postHookRoutingKey;
  private Channel postHookChannel;
  // TODO concurent map
  private HashMap<Long, Message> batch;
  private Optional<BatchTimeout> timout;
  // TODO flag to set either if to use batch ack, or if we have to keep track
  // TODO if each message is acknowledgable separately
  // batch process and ack
  // https://stackoverflow.com/questions/72315840/rabbitmq-consume-messages-in-batches-and-ack-them-all-at-once   
  private Boolean batchAck;

  ConsumerImpl(String queue, Channel channel) {
    super(channel);
    this.queue = queue;
    this.batchSize = 0;
    this.batch = new HashMap<Long, Message>();
    this.timout = Optional.empty();
  }

  public void setBatch(Integer size, Integer timeout) {
    this.batchSize = size;
    this.batchTimeout = timeout;
  }

  // TODO transform to generic function / callback
  public void setPostHook(Channel channel, String exchange, String routingKey) {
    this.postHookExchange = exchange;
    this.postHookRoutingKey = routingKey;
    this.postHookChannel = channel;
  }

  private void postHook(Envelope envelope, BasicProperties properties, byte[] body) throws IOException {
    if (this.postHookExchange.length() == 0 || this.postHookRoutingKey.length() == 0) {
      return;
    }

    this.postHookChannel.basicPublish(this.postHookExchange, this.postHookRoutingKey,
      MessageProperties.PERSISTENT_TEXT_PLAIN,
      this.getPostHookBodyEnvelope(envelope, properties, body));
  }

  private byte[] getPostHookBodyEnvelope(Envelope envelope, BasicProperties properties, byte[] body) {
    return body;
  }

  public void flushBatch() throws IOException {
    this.batch.forEach((key, m) -> {
      // TODO make foreach throw
      try {
        if (!m.acknowledgable) {
          this.getChannel().basicNack(key, false, true);
          System.out.println(" [x] Nacking " + new String(m.body, "UTF-8"));
          return;
        }
        this.postHook(m.envelope, m.properties, m.body);
        this.getChannel().basicAck(key, false);
        System.out.println(" [v] Acking " + new String(m.body, "UTF-8"));
      } catch (IOException ex) {
        System.out.println("could not flush tag " + key.toString() + ": " + ex.getMessage());
      }
    });
    this.timout.get().cancel();
    this.timout = Optional.empty();
    this.batch.clear();
    System.out.println(" [i] Flushing");
  }

  private ArrayList<Envelope> process(Envelope envelope, BasicProperties properties, byte[] body) {
    ArrayList<Envelope> processed = new ArrayList<Envelope>();
    // Do whatever transformation
    // use this.batch to search in other messages if aggregation is needed 
    // EG partial messages for flightdata

    // push in "processed" all envelopes of message which have been really processed and are therefore
    // ready to be acknowledgable.
    // 
    // ! in this example we ack onlt even messages
    if (this.batch.size() % 2 == 0) {
      processed.add(envelope);
    }

    if (this.batchSize != 0) {
      // put the message in the batch hashmap
      this.batch.put(envelope.getDeliveryTag(), new Message(envelope, properties, body));
    }

    return processed;
  }

  // https://www.rabbitmq.com/consumer-cancel.html
  @Override
  public void handleCancel(String consumerTag) throws IOException {
    System.out.println(" [x] Got cancelled '" + consumerTag + "'");
  }

  @Override
  public void handleDelivery(String consumerTag, Envelope envelope, BasicProperties properties, byte[] body) throws IOException {
    String message = new String(body, "UTF-8");
    System.out.println(" [v] Received '" + message + "'");

    ArrayList<Envelope> acknowledgable = this.process(envelope, properties, body);

    // Process batch
    if (this.batchSize != 0) {
      acknowledgable.forEach((e) -> this.batch.get(e.getDeliveryTag()).acknowledgable = true);

      // if exceed batch size, flush
      if (this.batch.size() == this.batchSize) {
        this.flushBatch();
      } else if (this.timout.isEmpty()) {
        // otherwise schedule timeout event 
        Timer t = new Timer();
        BatchTimeout task = new BatchTimeout(this);
        this.timout = Optional.of(task);
        t.schedule(task, new Date(System.currentTimeMillis() + this.batchTimeout));
      }
    } else {
      this.postHook(envelope, properties, body);
      this.getChannel().basicAck(envelope.getDeliveryTag(), false);
    }
  }

  @Override
  public void handleCancelOk(String consumerTag) {
    System.out.println(" [x] handleCancelOk '" + consumerTag + "'");
  }

  @Override
  public void handleConsumeOk(String consumerTag) {
    System.out.println(" [x] handleConsumeOk '" + consumerTag + "'");
  }

  @Override
  public void handleRecoverOk(String consumerTag) {
    System.out.println(" [x] handleRecoverOk '" + consumerTag + "'");
  }

  // thrown on precondition_failed: delivery acknowledgement
  // TODO re-call basicConsume on the consumer or recreate channel & consumer
  @Override
  public void handleShutdownSignal(String consumerTag, ShutdownSignalException sig) {
    System.out.println(" [x] handleShutdownSignal '" + consumerTag + "'");
  }
}
