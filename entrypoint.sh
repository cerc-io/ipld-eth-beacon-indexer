#!/bin/bash

sleep 10
echo "Starting ipld-eth-beacon-indexer"

echo /root/ipld-eth-beacon-indexer capture ${CAPTURE_MODE} --config /root/ipld-eth-beacon-config.json

/root/ipld-eth-beacon-indexer capture ${CAPTURE_MODE} --config /root/ipld-eth-beacon-config.json
rv=$?

if [ $rv != 0 ]; then
  echo "ipld-eth-beacon-indexer startup failed"
  echo 1 > /root/HEALTH
else
  echo "ipld-eth-beacon-indexer startup succeeded"
  echo 0 > /root/HEALTH
fi

tail -f /dev/null