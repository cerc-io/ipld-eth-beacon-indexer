package loghelper

import (
	log "github.com/sirupsen/logrus"
)

// A simple helper function that will help wrap the error message.
func LogEndpoint(endpoint string) *log.Entry {
	return log.WithFields(log.Fields{
		"endpoint": endpoint,
	})
}
