create extension if not exists postgis;
create database bdp;
\connect bdp
create user bdp with password :'BDP_PASSWORD'
create schema intimev2;
grant all privileges on schema intimev2 to bdp;
create user bdp_readonly with password :'BDP_READONLY_PASSWORD'
grant USAGE on schema intimev2 to bdp_readonly;
grant select on all tables in schema intimev2 to bdp_readonly;
alter default privileges in schema intimev2 grant select on all tables to bdp_readonly;