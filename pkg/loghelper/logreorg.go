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
