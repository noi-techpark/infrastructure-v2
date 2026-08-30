<!--
SPDX-FileCopyrightText: NOI Techpark <digital@noi.bz.it>

SPDX-License-Identifier: CC0-1.0
-->

# Cluster Management Runbooks

This document covers common cluster management procedures.

Before running any command, confirm your kubectl context is 
pointing to the right environment:
```sh
kubectl config current-context
```

## 1. Node Upgrade

Check current node versions:
```sh
kubectl get nodes -o wide
```

Cordon the node to prevent new pods from being scheduled on it.
Replace `<node-name>` with the name from the output above:
```sh
kubectl cordon <node-name>
```

Drain the node to safely move all running pods to other nodes:
```sh
kubectl drain <node-name> --ignore-daemonsets --delete-emptydir-data
```

Upgrade the node following the AWS node group update procedure
in the AWS console or via the AWS CLI.

Once the upgrade is complete, uncordon the node to allow pods 
to be scheduled on it again:
```sh
kubectl uncordon <node-name>
```

Verify the node is back and running:
```sh
kubectl get nodes -o wide
```

## 2. Rollback a Helm Release

Check the release history to find the revision you want to 
roll back to. Replace `<release-name>` with your Helm release 
name and `<namespace>` with the namespace it is deployed in 
(e.g. core, monitoring):
```sh
helm history <release-name> -n <namespace>
```

Rollback to the previous version:
```sh
helm rollback <release-name> -n <namespace>
```

Rollback to a specific revision number from the history above:
```sh
helm rollback <release-name> <revision-number> -n <namespace>
```

Verify the rollback was successful:
```sh
helm status <release-name> -n <namespace>
kubectl get pods -n <namespace>
```

## 3. Blue-Green Deployment

Blue-green deployment runs two identical environments side by 
side. The current live version is blue. The new version is 
green. Traffic is switched from blue to green once the new 
version is verified.

Deploy the new version. Replace `<green-deployment.yaml>` with 
the path to your deployment file:
```sh
kubectl apply -f <green-deployment.yaml>
```

Verify the new deployment is healthy. Replace `<namespace>` 
with your deployment namespace:
```sh
kubectl get pods -n <namespace> -l version=green
```

Switch all traffic to the new version by updating the service 
selector. Replace `<service-name>` and `<namespace>` with your 
actual values:
```sh
kubectl patch service <service-name> -n <namespace> \
  -p '{"spec":{"selector":{"version":"green"}}}'
```

Once the new version is confirmed stable, remove the old 
deployment. Replace `<blue-deployment-name>` with the name of 
your old deployment:
```sh
kubectl delete deployment <blue-deployment-name> -n <namespace>
```

## 4. Canary Deployment

Canary deployment gradually shifts traffic to a new version 
to reduce risk. Start with a small percentage of traffic and 
increase it as confidence grows.

Deploy the canary version alongside the stable version. 
Start with a low replica count to limit initial traffic exposure.
Replace `<canary-deployment.yaml>` with your deployment file:
```sh
kubectl apply -f <canary-deployment.yaml>
```

Monitor the canary logs to check for errors. Replace 
`<namespace>` with your namespace:
```sh
kubectl logs -n <namespace> -l version=canary --tail=100
```

If the canary looks stable, gradually shift replicas from 
stable to canary. The ratio controls traffic distribution 
(e.g. 5 canary + 5 stable = 50/50 split). Replace 
`<canary-deployment>`, `<stable-deployment>` and `<namespace>` 
with your actual values:
```sh
kubectl scale deployment <canary-deployment> -n <namespace> --replicas=5
kubectl scale deployment <stable-deployment> -n <namespace> --replicas=5
```

Once fully verified, complete the rollout:
```sh
kubectl scale deployment <canary-deployment> -n <namespace> --replicas=10
kubectl delete deployment <stable-deployment> -n <namespace>
```

If issues are found at any point, rollback the canary 
immediately:
```sh
kubectl scale deployment <canary-deployment> -n <namespace> --replicas=0
kubectl scale deployment <stable-deployment> -n <namespace> --replicas=10
```