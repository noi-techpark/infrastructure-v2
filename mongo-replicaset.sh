#!/usr/bin/env bash

# Referennce at https://www.mongodb.com/community/forums/t/docker-compose-replicasets-getaddrinfo-enotfound/14301/4
mongo --eval "rs.initiate({
          _id : 'rs0',
          members: [
            { _id : 0, host : 'mongodb1:27017' },
          ]
        })"

# it seems mongo can't connect to itself using docker service names
# therefore we have to launche the command to initiate the replication from outside the container
# docker exec odh-infrastructure-v2_mongodb1_1 mongo --eval "rs.initiate({
#            _id : 'rs0',
#            members: [
#              { _id : 0, host : 'mongodb1:27017' },
#            ]
#          })"