create database bdp;
\connect bdp
create extension postgis;
create user bdp with password :'bdp_pw';
create schema intimev2;
grant all privileges on schema intimev2 to bdp;
-- Unfortunately this is needed for initial flyway migration setup. Ideally revoke this after the DB is initialized, or never do it if you are starting from an existing dump
-- grant rds_superuser to bdp; 
create user bdp_readonly with password :'bdp_ro_pw';
grant USAGE on schema intimev2 to bdp_readonly;
alter default privileges in schema intimev2 grant select on tables to bdp_readonly;