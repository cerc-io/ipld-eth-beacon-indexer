// VulcanizeDB
// Copyright © 2022 Vulcanize

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.

// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.
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

// A simple herlper function to log slot and error.
func LogSlotError(slot uint64, err error) *log.Entry {
	return log.WithFields(log.Fields{
		"err":  err,
		"slot": slot,
	})
}

func LogSlotRangeError(startSlot uint64, endSlot uint64, err error) *log.Entry {
	return log.WithFields(log.Fields{
		"err":       err,
		"startSlot": startSlot,
		"endSlot":   endSlot,
	})
}
func LogSlotRangeStatementError(startSlot uint64, endSlot uint64, statement string, err error) *log.Entry {
	return log.WithFields(log.Fields{
		"err":          err,
		"startSlot":    startSlot,
		"endSlot":      endSlot,
		"SqlStatement": statement,
	})
}
