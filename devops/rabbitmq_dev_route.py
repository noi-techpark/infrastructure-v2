#!/usr/bin/env python3
import argparse
import sys
import subprocess
import json
import base64
import urllib.request
import urllib.error
import time

class SecretRetriever:
    def __init__(self, secret_pattern, namespace):
        self.secret_pattern = secret_pattern
        self.namespace = namespace

    def retrieve(self):
        secret_values = {}  # Initialize an empty dictionary to hold the decoded secrets
        try:
            # List all secrets in the namespace
            result = subprocess.run(["kubectl", "get", "secrets", "-n", self.namespace, "-o", "json"],
                                    check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            secrets = json.loads(result.stdout)

            # Filter secrets by name pattern and sort by creation timestamp
            matching_secrets = [secret for secret in secrets['items']
                                if self.secret_pattern in secret['metadata']['name']]
            if not matching_secrets:
                print("No secrets found matching the pattern.")
                return

            # Sort secrets by creation timestamp
            newest_secret = sorted(matching_secrets,
                                   key=lambda x: x['metadata']['creationTimestamp'],
                                   reverse=True)[0]  # Get the most recent secret

            # Assuming the secret data is base64 encoded, decode it
            print(f"Newest secret matching pattern '{self.secret_pattern}': {newest_secret['metadata']['name']}")
            # Decode each key-value pair and store in the dictionary
            for key, value in newest_secret['data'].items():
                decoded_value = base64.b64decode(value).decode('utf-8')
                secret_values[key] = decoded_value

            return secret_values  # Return the dictionary containing the decoded secrets

        except subprocess.CalledProcessError as e:
            print(f"Error retrieving secret: {e}", file=sys.stderr)
            sys.exit(1)
        except json.JSONDecodeError as e:
            print(f"Error decoding secret data: {e}", file=sys.stderr)
            sys.exit(1)
        except FileNotFoundError:
            print("kubectl is not installed or not found in PATH.", file=sys.stderr)
            sys.exit(1)

class CredentialPull:
    def __init__(self, profile, region = "eu-west-1"):
        self.profile = profile
        self.region = region

    def pull(self):
        command = ["aws", "eks", "--region", self.region, "update-kubeconfig", "--name", "aws-main-eu-01"]
        if self.profile not in [None, ""]:
            command += ["--profile", self.profile]
        
        try:
            subprocess.run(command, check=True)
            print("Kubeconfig updated successfully.")
        except subprocess.CalledProcessError as e:
            print("Failed to update kubeconfig:", e)
            sys.exit(1)
        except FileNotFoundError:
            print("aws CLI is not installed. Please install aws CLI to proceed.")
            sys.exit(1)

class RabbitMQConnector:
    def __init__(self, namespace, local_port, rabbitmq_user, rabbitmq_password):
        self.namespace = namespace
        self.local_port = local_port
        self.rabbitmq_user = rabbitmq_user
        self.rabbitmq_password = rabbitmq_password
        self.forwarding_process = None
        self.pod_name = None

    def get_pod_name(self, pod_name_pattern):
        # Command to get all pods in the namespace
        cmd = ["kubectl", "get", "pods", "-n", self.namespace, "-o", "json"]
        
        try:
            # Execute the command
            result = subprocess.run(cmd, check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            # Parse JSON output
            pods = json.loads(result.stdout)

            # Iterate over pods to find the first match
            for pod in pods["items"]:
                pod_name = pod["metadata"]["name"]
                if pod_name_pattern in pod_name:
                    self.pod_name = pod_name
                    return True
            return None  # No matching pod found
        except subprocess.CalledProcessError as e:
            print(f"Failed to get pods: {e}")
            return False
        except json.JSONDecodeError as e:
            print(f"Failed to parse JSON output: {e}")
            return False

    def start_port_forwarding(self):
        cmd = f"kubectl port-forward {self.pod_name} {self.local_port}:15672 -n {self.namespace}"
        self.forwarding_process = subprocess.Popen(cmd, shell=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        time.sleep(2)  # Give it a moment to establish the connection
        if self.forwarding_process.poll() is not None:
            raise Exception("Failed to start port forwarding. Check if the pod name and namespace are correct.")
        else:
            print(f"Port forwarding established with pod {self.pod_name} on local port {self.local_port}")

    def stop_port_forwarding(self):
        if self.forwarding_process:
            print("Stopping port forwarding...")
            if sys.platform == "win32":
                # Use taskkill on Windows to terminate the process group
                subprocess.call(['taskkill', '/F', '/T', '/PID', str(self.forwarding_process.pid)])
            else:
                # On Unix-like systems, you can use terminate()
                self.forwarding_process.terminate()
            self.forwarding_process.wait()
            print("Port forwarding stopped.")
            self.forwarding_process = None

    def create_queue_and_bind(self, vhost, queue_name, exchange_name, routing_key):
        vhost_encoded = urllib.parse.quote(vhost, safe='')
        # Step 1: Create Queue
        self.create_queue(vhost, queue_name)

        # Step 2: Create Binding
        url = f"http://localhost:{self.local_port}/api/bindings/{vhost_encoded}/e/{exchange_name}/q/{queue_name}"
        data = json.dumps({"routing_key": routing_key, "arguments": {}}).encode('utf-8')
        user_pass = base64.b64encode(f"{self.rabbitmq_user}:{self.rabbitmq_password}".encode('utf-8')).decode('utf-8')
        headers = {
            "Authorization": f"Basic {user_pass}",
            "Content-Type": "application/json"
        }

        req = urllib.request.Request(url, data=data, headers=headers, method='POST')
        try:
            with urllib.request.urlopen(req) as response:
                if response.status == 201 or response.status == 204:
                    print("Binding created successfully.")
                else:
                    print(f"Failed to create binding. Status code: {response.status}")
        except urllib.error.HTTPError as e:
            print(f"HTTP Error: {e.code} {e.reason}")
        except urllib.error.URLError as e:
            print(f"URL Error: {e.reason}")

    def create_queue(self, vhost, queue_name):
        vhost_encoded = urllib.parse.quote(vhost, safe='')
        url = f"http://localhost:{self.local_port}/api/queues/{vhost_encoded}/{queue_name}"
        data = json.dumps({"auto_delete": False, "durable": True, "arguments": {}}).encode('utf-8')
        user_pass = base64.b64encode(f"{self.rabbitmq_user}:{self.rabbitmq_password}".encode('utf-8')).decode('utf-8')
        headers = {
            "Authorization": f"Basic {user_pass}",
            "Content-Type": "application/json"
        }

        req = urllib.request.Request(url, data=data, headers=headers, method='PUT')
        try:
            with urllib.request.urlopen(req) as response:
                if response.status == 201 or response.status == 204:
                    print("Queue created successfully.")
                else:
                    print(f"Failed to create queue. Status code: {response.status}")
        except urllib.error.HTTPError as e:
            print(f"HTTP Error: {e.code} {e.reason}")
        except urllib.error.URLError as e:
            print(f"URL Error: {e.reason}")

    def delete_queue(self, vhost, queue_name):
        # URL-encode the vhost since it can contain characters like '/' that need encoding when used in URLs
        vhost_encoded = urllib.parse.quote(vhost, safe='')
        url = f"http://localhost:{self.local_port}/api/queues/{vhost_encoded}/{queue_name}"
        
        # Prepare the authorization header
        credentials = f"{self.rabbitmq_user}:{self.rabbitmq_password}"
        encoded_credentials = base64.b64encode(credentials.encode('utf-8')).decode('utf-8')
        headers = {
            "Authorization": f"Basic {encoded_credentials}"
        }

        # Create a request object with the method set to 'DELETE'
        req = urllib.request.Request(url, headers=headers, method='DELETE')
        
        try:
            # Perform the request
            with urllib.request.urlopen(req) as response:
                # Successful deletion usually returns a 204 No Content status
                if response.status == 204:
                    print("Queue deleted successfully.")
                else:
                    print(f"Unexpected response status: {response.status}")
        except urllib.error.HTTPError as e:
            print(f"HTTP Error: {e.code} {e.reason}")
        except urllib.error.URLError as e:
            print(f"URL Error: {e.reason}")

def check_cli_tools():
    # Check for kubectl
    try:
        subprocess.run(["kubectl", "version", "--client"], check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        print("kubectl is installed and OK.")
    except subprocess.CalledProcessError:
        print("kubectl command failed. Please ensure kubectl is installed and working.")
        sys.exit(1)
    except FileNotFoundError:
        print("kubectl is not installed. Please install kubectl to proceed.")
        sys.exit(1)

    # Check for aws cli
    try:
        subprocess.run(["aws", "--version"], check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        print("aws CLI is installed and OK.")
    except subprocess.CalledProcessError:
        print("aws command failed. Please ensure aws CLI is installed and working.")
        sys.exit(1)
    except FileNotFoundError:
        print("aws CLI is not installed. Please install aws CLI to proceed.")
        sys.exit(1)

def main():
    parser = argparse.ArgumentParser(description="Script to add or remove configurations.")

    # Adding subparsers for the add and remove commands
    subparsers = parser.add_subparsers(dest="command", required=True)

    # Add command
    add_parser = subparsers.add_parser("add", help="Add a new configuration")
    add_parser.add_argument("--queue", required=True, help="Queue name")
    add_parser.add_argument("--profile", help="AWS profile (optional)")
    add_parser.add_argument("--port", default="15672", help="Local port for forwarding (default: 15672)")
    add_parser.add_argument("--namespace", default="core", help="Kubernetes namespace (default: core)")
    add_parser.add_argument("--pod", default="rabbitmq", help="RabbitMQ pod (default: rabbitmq)")
    add_parser.add_argument("--region", default="eu-west-1", help="AWS region (default: eu-west-1)")
    add_parser.add_argument("--key", required=True, help="Routing key")

    # Remove command
    remove_parser = subparsers.add_parser("remove", help="Remove an existing configuration")
    remove_parser.add_argument("--queue", required=True, help="Queue name")
    remove_parser.add_argument("--profile", help="AWS profile (optional)")
    remove_parser.add_argument("--port", default="15672", help="Local port for forwarding (default: 15672)")
    remove_parser.add_argument("--namespace", default="core", help="Kubernetes namespace (default: core)")
    remove_parser.add_argument("--pod", default="rabbitmq", help="RabbitMQ pod (default: rabbitmq)")
    remove_parser.add_argument("--region", default="eu-west-1", help="AWS region (default: eu-west-1)")

    # Parse the arguments
    args = parser.parse_args()

    # Check if required CLI tools are installed
    check_cli_tools()  

    # Pull eks credentials:
    puller = CredentialPull(args.profile)
    puller.pull()

    # Pull rabbitmq secret
    secretManager = SecretRetriever("rabbitmq-svcbind", args.namespace)
    secrets = secretManager.retrieve()

    connector = RabbitMQConnector(args.namespace, args.port, secrets["username"], secrets["password"])
    if not connector.get_pod_name(args.pod):
        sys.exit(1)
        
    connector.start_port_forwarding()

    if args.command == "add":
        print(f"Adding configuration for queue: {args.queue} with routing_key {args.key}")
        ## TODO vhost should be configurable to ensure this script can't delete queues managed by other transformers
        connector.create_queue_and_bind("/", args.queue, "routed", args.key)
    elif args.command == "remove":
        print(f"Removing configuration for queue: {args.queue}")
        ## TODO vhost should be configurable to ensure this script can't delete queues managed by other transformers
        connector.delete_queue("/", args.queue)

    connector.stop_port_forwarding()

if __name__ == "__main__":
    main()
