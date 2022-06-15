#!/bin/bash

sleep 10
echo "Starting ipld-eth-beacon-indexer"

echo /root/ipld-eth-beacon-indexer capture ${CAPTURE_MODE} --config /root/ipld-eth-beacon-config.json > /root/ipld-eth-beacon-indexer.output

if [ ${CAPTURE_MODE} == "boot" ]; then
    /root/ipld-eth-beacon-indexer capture ${CAPTURE_MODE} --config /root/ipld-eth-beacon-config.json > /root/ipld-eth-beacon-indexer.output
    rv=$?

    if [ $rv != 0 ]; then
      echo "ipld-eth-beacon-indexer boot failed"
    else
      echo "ipld-eth-beacon-indexer boot succeeded"
    fi
      echo $rv > /root/HEALTH
      echo $rv
      cat /root/ipld-eth-beacon-indexer.output

    tail -f /dev/null
else
    exec /root/ipld-eth-beacon-indexer capture ${CAPTURE_MODE} --config /root/ipld-eth-beacon-config.json > /root/ipld-eth-beacon-indexer.output
fi