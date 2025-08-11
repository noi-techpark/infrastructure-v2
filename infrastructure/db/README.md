<!-- 
SPDX-FileCopyrightText: NOI Techpark <digital@noi.bz.it>

SPDX-License-Identifier: CC0-1.0
 -->

# What is this?
This is a database initialization script for the time series DB, to be run after the terraform creation of a new DB.
Just run `init_timeseries.sh` from it's directory and it will create the base database, schema and users.

After that, the next step is probably creating the postgres secret in k8s (see `docs/secrets.md`)

Once the flyway migration has gone through, revoke `rds_superuser` role from user `bdp`:
