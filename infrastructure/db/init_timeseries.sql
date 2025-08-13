-- SPDX-FileCopyrightText: NOI Techpark <digital@noi.bz.it>
--
-- SPDX-License-Identifier: CC0-1.0

-- drop database bdp;
create database bdp;
\connect bdp
create extension postgis;
create user bdp with password :'bdp_pw';
create schema intimev2;
grant all privileges on schema intimev2 to bdp;
create schema elaboration;
grant all privileges on schema elaboration to bdp;
-- Unfortunately this is needed for initial flyway migration setup. Ideally revoke this after the DB is initialized, or never do it if you are starting from an existing dump
grant rds_superuser to bdp; 
create user bdp_readonly with password :'bdp_ro_pw';
grant USAGE on schema intimev2 to bdp_readonly;
set session authorization bdp;
alter default privileges for user bdp in schema intimev2 grant select on tables to bdp_readonly;