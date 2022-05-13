package beaconclient

import (
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	dshelp "github.com/ipfs/go-ipfs-ds-help"
	"github.com/multiformats/go-multihash"
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
	return blockstore.BlockPrefix.String() + dbKey.String(), nil
}