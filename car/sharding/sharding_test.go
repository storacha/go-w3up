package sharding_test

import (
	"bytes"
	"crypto/rand"
	"io"
	"testing"

	"github.com/ipfs/go-cid"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/web3-storage/go-ucanto/core/ipld"
	"github.com/web3-storage/go-ucanto/core/ipld/block"
	"github.com/web3-storage/go-ucanto/core/ipld/hash/sha256"
	"github.com/web3-storage/go-ucanto/core/iterable"
	"github.com/storacha/go-w3up/car/sharding"
)

func randomRawBlock(t testing.TB, size int) ipld.Block {
	t.Helper()
	b := make([]byte, size)
	_, err := rand.Read(b)
	if err != nil {
		t.Fatal(err)
	}
	d, err := sha256.Hasher.Sum(b)
	if err != nil {
		t.Fatal(err)
	}
	l := cidlink.Link{Cid: cid.NewCidV1(0x55, d.Bytes())}
	return block.NewBlock(l, b)
}

func TestSharding(t *testing.T) {
	var roots []ipld.Link
	blocks := []ipld.Block{
		randomRawBlock(t, 4000),
		randomRawBlock(t, 4000),
	}
	iterator := iterable.From(blocks)
	size := 5000

	shards, err := sharding.NewSharder(roots, iterator, sharding.WithShardSize(size))
	if err != nil {
		t.Fatal(err)
	}

	var shdbufs [][]byte
	for {
		s, err := shards.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatal(err)
		}

		buf := new(bytes.Buffer)
		_, err = buf.ReadFrom(s)
		if err != nil {
			t.Fatal(err)
		}
		if len(buf.Bytes()) > size {
			t.Fatalf("shard was bigger than max size: %d > %d", len(buf.Bytes()), size)
		}
		shdbufs = append(shdbufs, buf.Bytes())
	}

	if len(shdbufs) != 2 {
		t.Fatalf("unexpected number of shards: %d", len(shdbufs))
	}
}
