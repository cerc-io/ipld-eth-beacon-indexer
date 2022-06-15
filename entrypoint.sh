#!/bin/bash

sleep 10
echo "Starting ipld-eth-beacon-indexer"

echo /root/ipld-eth-beacon-indexer capture ${CAPTURE_MODE} --config /root/ipld-eth-beacon-config.json > /root/ipld-eth-beacon-indexer.output

exec /root/ipld-eth-beacon-indexer capture ${CAPTURE_MODE} --config /root/ipld-eth-beacon-config.json > /root/ipld-eth-beacon-indexer.output
rv=$?

if [ $rv != 0 ]; then
  echo "ipld-eth-beacon-indexer failed"
  echo $rv > /root/HEALTH
  echo $rv
  cat /root/ipld-eth-beacon-indexer.output
else
  echo "ipld-eth-beacon-indexer succeeded"
  echo $rv > /root/HEALTH
  echo $rv
  cat /root/ipld-eth-beacon-indexer.output
fi

tail -f /dev/null