
#!/bin/bash

# start this in a separate terminal ideally
# kubectl port-forward -n core svc/rabbitmq-headless 15672 --address 0.0.0.0 &

# Connection details for rabbit and mongo. Run kubectl port-forward above beforehand
RABBIT_PW=`kubectl get secret -n collector rabbitmq-svcbind -o  jsonpath='{.data.password}' | base64 -d`

# download tool directly from endpoint so that we have matching version
wget -rq http://localhost:15672/cli/rabbitmqadmin
chmod +x rabbitmqadmin
radmin () {
    ./rabbitmqadmin -u opendatahub -p $RABBIT_PW $@
}

radmin declare queue name=test.detour
radmin declare binding source=routed destination=test.detour routing_key="echarging.route220"
radmin declare queue name=test.detour.target

radmin declare parameter component=shovel name=testshovel value='{"value":{"src-queue":"test.detour","dest-queue":"test.detour.target"}}'
