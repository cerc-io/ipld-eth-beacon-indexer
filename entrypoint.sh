#!/bin/bash
echo "Starting ipld-ethcl-indexer"

echo /root/ipld-ethcl-indexer capture head --db.address $DB_ADDRESS \
  --db.password $DB_PASSWORD \
  --db.port $DB_PORT \
  --db.username $DB_USER \
  --db.name $DB_NAME \
  --db.driver $DB_DRIVER \
  --bc.address $BC_ADDRESS \
  --bc.port $BC_PORT \
  --log.level $LOG_LEVEL

/root/ipld-ethcl-indexer capture head --db.address $DB_ADDRESS \
  --db.password $DB_PASSWORD \
  --db.port $DB_PORT \
  --db.username $DB_USER \
  --db.name $DB_NAME \
  --db.driver $DB_DRIVER \
  --bc.address $BC_ADDRESS \
  --bc.port $BC_PORT \
  --log.level $LOG_LEVEL

rv=$?

if [ $rv != 0 ]; then
  echo "ipld-ethcl-indexer startup failed"
  exit 1
fi

tail -f /dev/null