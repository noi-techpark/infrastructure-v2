# SPDX-FileCopyrightText: NOI Techpark <digital@noi.bz.it>
#
# SPDX-License-Identifier: CC0-1.0

#!/bin/bash

# terraform init so you have access to the state
# MAKE SURE YOU ARE USING THE CORRECT WORKSPACE
(cd ../terraform/db; terraform init)

# Extract the user credentials and database coordinates from terraform:
SECURITY_GROUP=`terraform -chdir=../terraform/db output -json | jq -r '.postgres_homeoffice_sg_id.value'`

# Retrieve current IP address
IP=`curl -s http://checkip.amazonaws.com`
aws ec2 authorize-security-group-ingress --group-id "$SECURITY_GROUP" --protocol tcp --port 5432  --cidr $IP/32 --profile default --output text --tag-specifications 'ResourceType=security-group-rule,Tags=[{Key=Name,Value=Clemens}]'
