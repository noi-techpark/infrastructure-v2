#!/usr/bin/env python

# SPDX-FileCopyrightText: NOI Techpark <digital@noi.bz.it>
#
# SPDX-License-Identifier: AGPL-3.0-or-later

import pika
# python -m pip install pika --upgrade

connection = pika.BlockingConnection(pika.ConnectionParameters('127.0.0.1'))
channel = connection.channel()

arguments={
  "x-dead-letter-exchange": "dlx_exchange", "x-dead-letter-routing-key": "dlx_key"}

args = {"alternate-exchange": "unroutable-exchange"}

channel.queue_declare(queue='skidata.carezza', durable=True)
channel.queue_declare(queue='skidata', durable=True)
channel.queue_declare(queue='mobility', durable=True)

channel.queue_bind(queue="skidata.carezza", exchange="routed", routing_key="skidata.carezza")
channel.queue_bind(queue="skidata", exchange="routed", routing_key="skidata.#")
channel.queue_bind(queue="mobility", exchange="routed", routing_key="mobility.*")
