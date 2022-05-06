package loghelper

import (
	log "github.com/sirupsen/logrus"
)

// A simple helper function that will help wrap the reorg error messages.
func LogReorgError(slot string, latestBlockRoot string, err error) *log.Entry {
	return log.WithFields(log.Fields{
		"err":             err,
		"slot":            slot,
		"latestBlockRoot": latestBlockRoot,
	})
}

// A simple helper function that will help wrap regular reorg messages.
func LogReorg(slot string, latestBlockRoot string) *log.Entry {
	return log.WithFields(log.Fields{
		"slot":            slot,
		"latestBlockRoot": latestBlockRoot,
	})
}
