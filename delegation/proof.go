package delegation

import (
	"bytes"
	"fmt"
	"io"

	"github.com/web3-storage/go-ucanto/core/car"
	"github.com/web3-storage/go-ucanto/core/dag/blockstore"
	"github.com/web3-storage/go-ucanto/core/delegation"
	"github.com/web3-storage/go-ucanto/core/ipld/block"
)

// ExtractProof is a temporary helper to extract a proof from a proof archive
// with fallback to extracting from the legacy encoding.
//
// It will first attempt to extract using `delegation.Extract` from
// `go-ucanto/core/delegation` and falls back to decoding by reading a plain
// CAR file, assuming the last block is the delegation root.
func ExtractProof(b []byte) (delegation.Delegation, error) {
	proof, err := delegation.Extract(b)
	if err != nil {
		// try decode legacy format
		_, blocks, err := car.Decode(bytes.NewReader(b))
		if err != nil {
			return nil, fmt.Errorf("extracting proof: %s", err)
		}

		var rt block.Block
		bs, err := blockstore.NewBlockStore()
		if err != nil {
			return nil, fmt.Errorf("creating blockstore: %s", err)
		}
		for {
			bl, err := blocks.Next()
			if err != nil {
				if err == io.EOF {
					break
				}
				return nil, fmt.Errorf("reading block: %s", err)
			}
			err = bs.Put(bl)
			if err != nil {
				return nil, fmt.Errorf("putting block: %s", err)
			}
			rt = bl
		}

		proof = delegation.NewDelegation(rt, bs)
	}

	return proof, nil
}
