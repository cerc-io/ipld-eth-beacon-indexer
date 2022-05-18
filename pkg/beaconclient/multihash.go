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
package beaconclient

import (
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	dshelp "github.com/ipfs/go-ipfs-ds-help"
	"github.com/multiformats/go-multihash"
	log "github.com/sirupsen/logrus"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/loghelper"
)

const SSZ_SHA2_256_PREFIX uint64 = 0xb501

// MultihashKeyFromSSZRoot converts a SSZ-SHA2-256 root hash into a blockstore prefixed multihash key
func MultihashKeyFromSSZRoot(root []byte) (string, error) {
	mh, err := multihash.Encode(root, SSZ_SHA2_256_PREFIX)
	if err != nil {
		loghelper.LogError(err).Error("Unable to create a multihash Key")
		return "", err
	}
	dbKey := dshelp.MultihashToDsKey(mh)
	mhKey := blockstore.BlockPrefix.String() + dbKey.String()
	log.WithFields(log.Fields{"mhKey": mhKey, "len": len(root)}).Debug("The MHKEY")
	return mhKey, nil
}
