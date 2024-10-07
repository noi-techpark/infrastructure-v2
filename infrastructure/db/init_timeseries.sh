#!/bin/bash

# terraform init so you have access to the state
# MAKE SURE YOU ARE USING THE CORRECT WORKSPACE
(cd infrastructure/terraform/db; terraform init)

# Extract the user credentials and database coordinates from terraform:
EXTRACTED_JSON=`terraform -chdir=infrastructure/terraform/db output -json | jq -r '{hostname: .odh_postgres_hostname.value, port: .odh_postgres_port.value, usr_root: .odh_postgres_username, pw_root: .odh_postgres_password, pw_readwrite: .odh_postgres_password_bdp.value, pw_readonly: .odh_postgres_password_bdp_readonly.value}'`

# Get the value from the extracted json. You can supply your values in another way if you didn't use the terraform script
POSTGRES_HOST=`jq '.hostname' -r <<< "$EXTRACTED_JSON"`
POSTGRES_PORT=`jq '.port' -r <<< "$EXTRACTED_JSON"`
POSTGRES_USR=`jq '.usr_root' -r <<< "$EXTRACTED_JSON"`
POSTGRES_PW='.pw_root' -r <<< "$EXTRACTED_JSON"`
POSTGRES_R_PW=`jq '.pw_readonly' -r <<< "$EXTRACTED_JSON"`
POSTGRES_RW_PW=`jq '.pw_readwrite' -r <<< "$EXTRACTED_JSON"`