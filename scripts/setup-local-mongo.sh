#!/bin/bash

## https://gist.github.com/harveyconnor/518e088bad23a273cae6ba7fc4643549

MONGODB1=db-01.hydra.local
MONGODB2=db-02.hydra.local
MONGODB3=db-arbiter.hydra.local

echo "**********************************************" ${MONGODB1}
echo "Waiting for startup.."
until curl http://${MONGODB1}:27017/serverStatus\?text\=1 2>&1 | grep uptime | head -1; do
  printf '.'
  sleep 1
done

echo SETUP.sh time now: `date +"%T" `
mongosh --host ${MONGODB1}:27017 <<EOF

var initCfg = {
    "_id": "rs0",
    "version": 1,
    "members": [
      {
        "_id": 0,
        "host": "${MONGODB1}:27017",
        "arbiterOnly": false
      },
      {
        "_id": 1,
        "host": "${MONGODB2}:27017",
        "arbiterOnly": false
      },
      {
        "_id": 2,
        "host": "${MONGODB3}:27017",
        "arbiterOnly": true
      }
    ],
    "settings": { "chainingAllowed": true }
};

rs.initiate(initCfg, { force: true });

var cfg = {
    "_id": "rs0",
    "protocolVersion": 1,
    "version": 1,
    "members": [
      {
        "_id": 0,
        "host": "${MONGODB1}:27017",
        "priority": 2,
        "arbiterOnly": false
      },
      {
        "_id": 1,
        "host": "${MONGODB2}:27017",
        "priority": 0,
        "arbiterOnly": false
      },
      {
        "_id": 2,
        "host": "${MONGODB3}:27017",
        "priority": 0,
        "arbiterOnly": true
      }
    ],
    "settings": { "chainingAllowed": true }
};
rs.reconfig(cfg, { force: true });

db.getMongo().setReadPref('nearest');
db.getMongo().setReadPref();
EOF
