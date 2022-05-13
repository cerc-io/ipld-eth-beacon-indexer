#!/bin/bash

sleep 10
echo "Starting ipld-ethcl-indexer"

echo /root/ipld-ethcl-indexer capture ${CAPTURE_MODE} --db.address $DB_ADDRESS \
  --db.password $DB_PASSWORD \
  --db.port $DB_PORT \
  --db.username $DB_USER \
  --db.name $DB_NAME \
  --db.driver $DB_DRIVER \
  --bc.address $BC_ADDRESS \
  --bc.port $BC_PORT \
  --log.level $LOG_LEVEL\
  --t.skipSync=$SKIP_SYNC \
  --kg.increment $KNOWN_GAP_INCREMENT

/root/ipld-ethcl-indexer capture ${CAPTURE_MODE} --db.address $DB_ADDRESS \
  --db.password $DB_PASSWORD \
  --db.port $DB_PORT \
  --db.username $DB_USER \
  --db.name $DB_NAME \
  --db.driver $DB_DRIVER \
  --bc.address $BC_ADDRESS \
  --bc.port $BC_PORT \
  --log.level $LOG_LEVEL \
  --t.skipSync=$SKIP_SYNC \
  --kg.increment $KNOWN_GAP_INCREMENT

rv=$?

if [ $rv != 0 ]; then
  echo "ipld-ethcl-indexer startup failed"
  echo 1 > /root/HEALTH
else
  echo "ipld-ethcl-indexer startup succeeded"
  echo 0 > /root/HEALTH
fi

tail -f /dev/null