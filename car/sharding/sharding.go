package sharding

import (
	"fmt"
	"io"

	"github.com/multiformats/go-varint"
	"github.com/storacha/go-ucanto/core/car"
	"github.com/storacha/go-ucanto/core/ipld"
	"github.com/storacha/go-ucanto/core/ipld/block"
	"github.com/storacha/go-ucanto/core/ipld/codec/cbor"
	"github.com/storacha/go-ucanto/core/iterable"
)

// https://observablehq.com/@gozala/w3up-shard-size
const ShardSize = 133_169_152

/** Byte length of a CBOR encoded CAR header with zero roots. */
const noRootsHeaderLen = 17

// Option is an option configuring a sharder.
type Option func(cfg *sharderConfig) error

type sharderConfig struct {
	shdsize int
}

// WithShardSize configures the size of the shards - default 133,169,152 bytes.
func WithShardSize(size int) Option {
	return func(cfg *sharderConfig) error {
		cfg.shdsize = size
		return nil
	}
}

func NewSharderFromCAR(reader io.Reader) (iterable.Iterator[io.Reader], error) {
	roots, blocks, err := car.Decode(reader)
	if err != nil {
		return nil, fmt.Errorf("decoding CAR: %s", err)
	}
	return NewSharder(roots, blocks)
}

func NewSharder(roots []ipld.Link, blocks iterable.Iterator[block.Block], options ...Option) (iterable.Iterator[io.Reader], error) {
	cfg := sharderConfig{shdsize: ShardSize}
	for _, opt := range options {
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}

	hdrlen, err := headerEncodingLength(roots)
	if err != nil {
		return nil, fmt.Errorf("encoding header: %s", err)
	}

	maxblklen := cfg.shdsize - hdrlen

	nxt, err := blocks.Next()
	if err != nil {
		return nil, err
	}

	return iterable.NewIterator(func() (io.Reader, error) {
		if nxt == nil {
			return nil, io.EOF
		}

		clen := 0
		return car.Encode(roots, iterable.NewIterator(func() (block.Block, error) {
			var blk ipld.Block
			if nxt != nil {
				blk = nxt
				nxt = nil
			} else {
				blk, err = blocks.Next()
				if err != nil {
					return nil, err
				}
			}

			blklen := blockEncodingLength(blk)
			if blklen > maxblklen {
				return nil, fmt.Errorf("block will cause CAR to exceed shard size: %s", blk.Link())
			}

			if clen+blklen > maxblklen {
				nxt = blk
				return nil, io.EOF
			}

			clen += blklen
			return blk, nil
		})), nil
	}), nil
}

// func NewSharder(blocks iterable.Iterator[block.Block], options ...Option) (iterable.Iterator[io.Reader], error) {
// 	cfg := sharderConfig{shdsize: shardSize}
// 	for _, opt := range options {
// 		if err := opt(&cfg); err != nil {
// 			return nil, err
// 		}
// 	}

// 	maxblklen := cfg.shdsize - noRootsHeaderLen
// 	var shdblks []block.Block
// 	var rdyblks []block.Block
// 	clen := 0

// 	return iterable.NewIterator(func() (io.Reader, error) {
// 		for {
// 			if rdyblks != nil {
// 				shd := car.Encode([]ipld.Link{}, iterable.From(rdyblks))
// 				rdyblks = nil
// 				return shd, nil
// 			}

// 			blk, err := blocks.Next()
// 			if err != nil {
// 				if err == io.EOF {
// 					if len(shdblks) == 0 {
// 						return nil, io.EOF
// 					}

// 					roots := []ipld.Link{shdblks[len(shdblks)-1].Link()}

// 					hdrlen, err := headerEncodingLength(roots)
// 					if err != nil {
// 						return nil, fmt.Errorf("encoding header: %s", err)
// 					}

// 					// if adding CAR root overflows the shard limit we move overflowing
// 					// blocks into another CAR.
// 					if hdrlen+clen > shardSize {
// 						overage := hdrlen + clen - shardSize
// 						oblks := []block.Block{}
// 						olen := 0
// 						for olen < overage {
// 							blk := shdblks[len(shdblks)-1]
// 							shdblks = shdblks[0 : len(shdblks)-1]
// 							oblks = append([]block.Block{blk}, oblks...)
// 							olen += blockEncodingLength(blk)

// 							// need at least 1 block in original shard
// 							if len(shdblks) < 1 {
// 								return nil, fmt.Errorf("block will cause CAR to exceed shard size: %s", blk.Link())
// 							}
// 						}
// 						shd := car.Encode([]ipld.Link{}, iterable.From(shdblks))
// 						// next time we will return the overflow shards
// 						shdblks = oblks
// 						return shd, nil
// 					} else {
// 						shd := car.Encode(roots, iterable.From(shdblks))
// 						shdblks = nil
// 						return shd, nil
// 					}
// 				}
// 				return nil, err
// 			}

// 			blklen := blockEncodingLength(blk)
// 			if blklen > maxblklen {
// 				return nil, fmt.Errorf("block will cause CAR to exceed shard size: %s", blk.Link())
// 			}

// 			if len(shdblks) > 0 && clen+blklen > maxblklen {
// 				rdyblks = shdblks
// 				shdblks = []block.Block{}
// 				clen = 0
// 			}
// 			shdblks = append(shdblks, blk)
// 			clen += blklen
// 		}
// 	}), nil
// }

type header struct {
	version uint64
	roots   []ipld.Link
}

func headerEncodingLength(roots []ipld.Link) (int, error) {
	if len(roots) == 0 {
		return noRootsHeaderLen, nil
	}

	b, err := cbor.Encode(&header{1, roots}, nil)
	if err != nil {
		return 0, err
	}

	hdlen := len(b)
	vilen := varint.UvarintSize(uint64(hdlen))
	return hdlen + vilen, nil
}

func blockEncodingLength(block block.Block) int {
	pllen := len(block.Link().Binary()) + len(block.Bytes())
	vilen := varint.UvarintSize(uint64(pllen))
	return pllen + vilen
}
