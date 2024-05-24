
#!/bin/bash

# https://rawcdn.githack.com/rabbitmq/rabbitmq-server/v3.13.2/deps/rabbitmq_management/priv/www/api/index.html

# start this in a separate terminal ideally
# kubectl port-forward -n core svc/rabbitmq-headless 15672 --address 0.0.0.0 &

# download tool directly from endpoint so that we have matching version
curl -O -q http://localhost:15672/cli/rabbitmqadmin
chmod +x rabbitmqadmin
RABBIT_PW=`kubectl get secret -n collector rabbitmq-svcbind -o  jsonpath='{.data.password}' | base64 -d`
radmin () {
    ./rabbitmqadmin -u opendatahub -p $RABBIT_PW $@
}

srcroute=echarging.route220
detourroute=$srcroute.detour

# radmin declare queue name=$srcroute.detour
# radmin declare binding source=routed destination=$detourroute routing_key=$srcroute
# shoveljson='{"src-uri":"amqp://","src-queue":"'$srcroute'","dest-uri":"amqp://","dest-queue":"'$detourroute'","src-delete-after":"queue-length"}'
# radmin declare parameter component=shovel name=testshovel value=$shoveljson

# radmin delete parameter component=shovel name=testshovel
# radmin delete queue name=test.detour
# radmin delete queue name=test.detour.target
