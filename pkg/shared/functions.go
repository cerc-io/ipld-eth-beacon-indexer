package shared

import (
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	dshelp "github.com/ipfs/go-ipfs-ds-help"
	"github.com/multiformats/go-multihash"
)

const SSZ_SHA2_256_PREFIX int = 0xb501

// MultihashKeyFromSSZRoot converts a SSZ-SHA2-256 root hash into a blockstore prefixed multihash key
func MultihashKeyFromSSZRoot(root []byte) (string, error) {
	mh, err := multihash.Encode(root, SSZ_SHA2_256_PREFIX)
	if err != nil {
		return "", err
	}
	dbKey := dshelp.MultihashToDsKey(mh)
	return blockstore.BlockPrefix.String() + dbKey.String(), nil
}
