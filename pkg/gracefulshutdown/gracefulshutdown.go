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
package gracefulshutdown

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vulcanize/ipld-eth-beacon-indexer/pkg/loghelper"
)

// operation is a clean up function on shutting down
type Operation func(ctx context.Context) error

var (
	TimeoutErr = func(timeout string) error {
		return fmt.Errorf("The Timeout %s, has been elapsed, the application will forcefully exit", timeout)
	}
)

// gracefulShutdown waits for termination syscalls and doing clean up operations after received it
func Shutdown(ctx context.Context, notifierCh chan os.Signal, timeout time.Duration, ops map[string]Operation) (<-chan struct{}, <-chan error) {
	waitCh := make(chan struct{})
	errCh := make(chan error)
	go func() {

		// add any other syscalls that you want to be notified with
		signal.Notify(notifierCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
		// Wait for one or the other...
		select {
		case <-notifierCh:
		case <-ctx.Done():
		}

		log.Info("Shutting Down your application")

		// set timeout for the ops to be done to prevent system hang
		timeoutFunc := time.AfterFunc(timeout, func() {
			log.Warnf(TimeoutErr(timeout.String()).Error())
			errCh <- TimeoutErr(timeout.String())
		})

		defer timeoutFunc.Stop()

		var wg sync.WaitGroup

		// Do the operations asynchronously to save time
		for key, op := range ops {
			wg.Add(1)
			innerOp := op
			innerKey := key
			go func() {
				defer wg.Done()

				log.Infof("cleaning up: %s", innerKey)
				if err := innerOp(ctx); err != nil {
					loghelper.LogError(err).Errorf("%s: clean up failed: %s", innerKey, err.Error())
					return
				}

				log.Infof("%s was shutdown gracefully", innerKey)
			}()
		}

		wg.Wait()

		close(waitCh)
	}()

	return waitCh, errCh
}
