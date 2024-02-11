# Remote RabbitMQ Queue Manager

This documentation covers the `rabbitmq_dev_route.py` script located in the `devops` directory of the Open Data Hub project. The script is designed to manage RabbitMQ configurations, specifically adding or 
removing configurations as needed.

This script helps developers to create queues to route messages for yet-to-be-implemented transformers without the need for manually connecting to the RabbitMQ control plane. It automates the process of configuring queue routing, making it easier to integrate new services and components into the Open Data Hub's data processing pipeline.

## System Requirements

- **Python Version:** Python3
- **Dependencies:** None

## Overview

The `rabbitmq_dev_route.py` script enables users to add or remove configurations to RabbitMQ within the Open Data Hub ecosystem. It provides a command-line interface for specifying the queue operations, such as adding or removing a queue configuration. The script interacts with AWS (optional) for credentials, Kubernetes for namespace and pod information, and RabbitMQ for queue management. It ensures secure connection and operation by retrieving necessary secrets for RabbitMQ.

## Usage

To use the script, navigate to the script's directory and execute it with Python3. The script offers two main commands: `add` for adding a new configuration and `remove` for removing an existing configuration.

### Adding a Configuration

```bash
python3 rabbitmq_dev_route.py add --queue QUEUE_NAME --key ROUTING_KEY [OPTIONS]
```

- **--queue**: Name of the queue to add (required).
- **--key**: Routing key associated with the queue (required).
- **--profile**: AWS profile name (optional). If specified, uses this AWS profile for credentials.
- **--port**: Local port for forwarding (default: 15672).
- **--namespace**: Kubernetes namespace (default: core).
- **--pod**: RabbitMQ pod (default: rabbitmq).
- **--region**: AWS region (default: eu-west-1).

### Removing a Configuration
```bash
python3 rabbitmq_dev_route.py remove --queue QUEUE_NAME [OPTIONS]
```

- **--queue**: Name of the queue to remove (required).
- **--profile**: AWS profile name (optional). If specified, uses this AWS profile for credentials.
- **--port**: Local port for forwarding (default: 15672).
- **--namespace**: Kubernetes namespace (default: core).
- **--pod**: RabbitMQ pod (default: rabbitmq).
- **--region**: AWS region (default: eu-west-1).

## Additional Information
The script checks for required CLI tools and pulls necessary credentials from AWS and Kubernetes to interact with RabbitMQ securely.
It manages connections to RabbitMQ by starting and stopping port forwarding as needed during the add or remove operations.
When adding a configuration, the script creates a queue and binds it with the provided routing key. When removing a configuration, it deletes the specified queue.