package main

import (
	"fmt"
	"io"
	"log"
	"net/url"
	"os"

	uclient "github.com/storacha/go-ucanto/client"
	"github.com/storacha/go-ucanto/core/car"
	"github.com/storacha/go-ucanto/core/delegation"
	"github.com/storacha/go-ucanto/core/ipld"
	"github.com/storacha/go-ucanto/core/result"
	"github.com/storacha/go-ucanto/did"
	"github.com/storacha/go-ucanto/principal"
	"github.com/storacha/go-w3up/capability/uploadadd"
	"github.com/storacha/go-w3up/car/sharding"
	"github.com/storacha/go-w3up/client"
	"github.com/storacha/go-w3up/cmd/util"
	"github.com/urfave/cli/v2"
)

// upload handles file and directory uploads to Storacha
func upload(cCtx *cli.Context) error {
	signer := util.MustGetSigner()
	conn := util.MustGetConnection()
	space := util.MustParseDID(cCtx.String("space"))
	proofs := []delegation.Delegation{util.MustGetProof(cCtx.String("proof"))}
	receiptsURL := util.MustGetReceiptsURL()

	// Handle options
	isCAR := cCtx.String("car") != ""
	isJSON := cCtx.Bool("json")
	// isVerbose := cCtx.Bool("verbose")
	isWrap := cCtx.Bool("wrap")
	// shardSize := cCtx.Int("shard-size")
	// concurrentRequests := cCtx.Int("concurrent-requests")

	var paths []string
	if isCAR {
		paths = []string{cCtx.String("car")}
	} else {
		paths = cCtx.Args().Slice()
	}

	var root ipld.Link
	if isCAR {
		var err error
		root, err = uploadCAR(paths[0], signer, conn, space, proofs, receiptsURL)
		if err != nil {
			return err
		}
	} else {
		if len(paths) == 1 && !isWrap {
			var err error
			root, err = uploadFile(paths[0], signer, conn, space, proofs, receiptsURL)
			if err != nil {
				return err
			}
		} else {
			var err error
			root, err = uploadDirectory(paths, signer, conn, space, proofs, receiptsURL)
			if err != nil {
				return err
			}
		}
	}

	if isJSON {
		fmt.Printf("{\"root\":\"%s\"}\n", root)
	} else {
		fmt.Printf("⁂ https://w3s.link/ipfs/%s\n", root)
	}

	return nil
}

func uploadCAR(path string, signer principal.Signer, conn uclient.Connection, space did.DID, proofs []delegation.Delegation, receiptsURL *url.URL) (ipld.Link, error) {
	f0, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening file: %s", err)
	}
	defer f0.Close()

	var shdlnks []ipld.Link

	stat, err := f0.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat file: %s", err)
	}

	if stat.Size() < sharding.ShardSize {
		link, err := addBlob(f0, signer, conn, space, proofs, receiptsURL)
		if err != nil {
			return nil, err
		}

		shdlnks = append(shdlnks, link)
	} else {
		_, blocks, err := car.Decode(f0)
		if err != nil {
			log.Fatalf("decoding CAR: %s", err)
		}
		shds, err := sharding.NewSharder([]ipld.Link{}, blocks)
		if err != nil {
			log.Fatalf("sharding CAR: %s", err)
		}

		for shd, err := range shds {
			if err != nil {
				log.Fatal(err)
			}

			link, err := addBlob(shd, signer, conn, space, proofs, receiptsURL)
			if err != nil {
				return nil, err
			}

			shdlnks = append(shdlnks, link)
		}
	}

	// TODO: build, add and register index

	f1, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening file: %s", err)
	}
	defer f1.Close()

	roots, _, err := car.Decode(f1)
	if err != nil {
		return nil, fmt.Errorf("reading roots: %s", err)
	}

	if len(roots) == 0 {
		return nil, fmt.Errorf("missing root CID")
	}

	rcpt, err := client.UploadAdd(
		signer,
		space,
		uploadadd.Caveat{
			Root:   roots[0],
			Shards: shdlnks,
		},
		client.WithConnection(conn),
		client.WithProofs(proofs),
	)
	if err != nil {
		return nil, err
	}

	_, upFailure := result.Unwrap(rcpt.Out())
	if upFailure != nil {
		return nil, fmt.Errorf("%+v\n", upFailure)
	}

	return roots[0], nil
}

func uploadFile(path string, signer principal.Signer, conn uclient.Connection, space did.DID, proofs []delegation.Delegation, receiptsURL *url.URL) (ipld.Link, error) {
	return nil, nil
}

func uploadDirectory(paths []string, signer principal.Signer, conn uclient.Connection, space did.DID, proofs []delegation.Delegation, receiptsURL *url.URL) (ipld.Link, error) {
	return nil, nil
}

func addBlob(content io.Reader, signer principal.Signer, conn uclient.Connection, space did.DID, proofs []delegation.Delegation, receiptsURL *url.URL) (ipld.Link, error) {
	contentLink, _, err := client.BlobAdd(content, signer, space, receiptsURL, client.WithConnection(conn), client.WithProofs(proofs))
	if err != nil {
		return nil, err
	}

	return contentLink, nil
}
