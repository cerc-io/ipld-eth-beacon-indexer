// A simple function to help with logging errors.
package loghelper

import (
	log "github.com/sirupsen/logrus"
)

// A simple helper function that will help wrap the error message.
func LogError(err error) *log.Entry {
	return log.WithFields(log.Fields{
		"err": err,
	})
}
