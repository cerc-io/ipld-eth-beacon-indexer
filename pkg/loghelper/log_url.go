package loghelper

import (
	log "github.com/sirupsen/logrus"
)

// A simple helper function that will help wrap the error message.
func LogUrl(url string) *log.Entry {
	return log.WithFields(log.Fields{
		"url": url,
	})
}
