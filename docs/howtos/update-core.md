# Update Core Components

## Breaking Changes Flow

Sometimes some breaking changes in the core components of the infrastructure (router, writer, ...) have to be made.  
In that cases we need to follow a procedure to ensure no messages are lost.

### 1. Set `writer` replica count to 0
This ensures all messages coming from collectors stop in the `ingress` queue and are not lost.

```
kubectl scale deployment -n core  raw-writer --replicas 0
```

### 2. Deploy changes
Deploy the new components **starting from the furthest from ingress**

```
data-bridge > raw-data-table > router > writer
```

This ensures we don't accidentally process messages with an old standard or old logic, making them difficult to trace and reprocess.

### 3. Reset Bindings (optiona)
If the changes involved `routed` exchange (IE changed the declaration) and we need to delete the exchange, we need to rest all transformers bindings.

1. Send a proper message (with the expected format) in the `ready` exchange so that the router can recreate the `routed exchange`
2. Restart all transformers to rebind their queues to the exchange:
```
kubectl get pods -n collector --no-headers | grep '^tr-' | awk '{print $1}' | xargs -r -I {} kubectl delete pod {} -n collector
```

### 4. Restart or redeploy the Writer

#### Restart:
```
kubectl scale deployment -n core  raw-writer --replicas 1
```