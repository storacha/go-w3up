package client

import (
	"fmt"

	"github.com/web3-storage/go-ucanto/client"
	"github.com/web3-storage/go-ucanto/core/invocation"
	"github.com/web3-storage/go-ucanto/core/receipt"
	"github.com/web3-storage/go-ucanto/did"
	"github.com/web3-storage/go-ucanto/principal"
	"github.com/web3-storage/go-w3up/capability/uploadlist"
)

func List(signer principal.Signer, space did.DID, params *uploadlist.Caveat, options ...Option) (receipt.Receipt[*uploadlist.UploadListSuccess, *uploadlist.UploadListFailure], error) {
	cfg := ClientConfig{conn: DefaultConnection}
	for _, opt := range options {
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}

	inv, err := invocation.Invoke(
		signer,
		cfg.conn.ID(),
		uploadlist.NewCapability(space, &uploadlist.Caveat{}),
		convertToInvocationOptions(cfg)...,
	)
	if err != nil {
		return nil, err
	}

	resp, err := client.Execute([]invocation.Invocation{inv}, cfg.conn)
	if err != nil {
		return nil, err
	}

	rcptlnk, ok := resp.Get(inv.Link())
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("receipt not found: %s", inv.Link())
	}

	reader, err := uploadlist.NewReceiptReader()
	if err != nil {
		return nil, err
	}

	return reader.Read(rcptlnk, resp.Blocks())
}
