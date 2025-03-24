package car

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"

	"github.com/goddhi/storacha-go/dag/adder"
	"github.com/ipfs/boxo/blockservice"
	blockstore "github.com/ipfs/boxo/blockstore"
	"github.com/ipfs/boxo/ipld/merkledag"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/sync"
	gocar "github.com/ipld/go-car/v2"
	carstorage "github.com/ipld/go-car/v2/storage"
)

func BuildCAR(ctx context.Context, file fs.File, path string, fsys fs.FS) (io.Reader, cid.Cid, error) {
	ds := datastore.NewMapDatastore()
	bs := blockstore.NewBlockstore(sync.MutexWrap(ds))
	bserv := blockservice.New(bs, nil)
	dagService := merkledag.NewDAGService(bserv)

	add, err := adder.CreateNewAdder(ctx, dagService)
	if err != nil {
		return nil, cid.Undef, fmt.Errorf("creating adder: %w", err)
	}
	defer add.Close()

	rootCid, err := add.Add(file, path, fsys)
	if err != nil {
		return nil, cid.Undef, fmt.Errorf("adding to DAG: %w", err)
	}

	buf := new(bytes.Buffer)

	carBlockStore, err := carstorage.NewWritable(buf, []cid.Cid{rootCid}, gocar.WriteAsCarV1(true))
	if err != nil {
		return nil, cid.Undef, fmt.Errorf("creating CAR blockstore: %w", err)
	}
	defer carBlockStore.Finalize() 

	seenBlocks := make(map[string]bool)

	var collectBlocks func(c cid.Cid) error
	collectBlocks = func(c cid.Cid) error {
		if seenBlocks[c.String()] {
			return nil
		}
		seenBlocks[c.String()] = true

		nd, err := dagService.Get(ctx, c)
		if err != nil {
			return fmt.Errorf("getting node %s: %w", c, err)
		}

		blk, err := blocks.NewBlockWithCid(nd.RawData(), c)
		if err != nil {
			return fmt.Errorf("creating block: %w", err)
		}

		err = carBlockStore.Put(ctx, blk.Cid().String(), blk.RawData())
		if err != nil {
			return fmt.Errorf("writing block to CAR: %w", err)
		}

		for _, link := range nd.Links() {
			err := collectBlocks(link.Cid)
			if err != nil {
				return err
			} 
		}
		return nil
	}

	err = collectBlocks(rootCid)
	if err != nil {
		return nil, cid.Undef, fmt.Errorf("collecting blocks: %w", err)
	}

	return buf, rootCid, nil
}
