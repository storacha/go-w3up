package client

import (
	"fmt"

	"github.com/web3-storage/go-ucanto/client"
	"github.com/web3-storage/go-ucanto/core/invocation"
	"github.com/web3-storage/go-ucanto/core/receipt"
	"github.com/web3-storage/go-ucanto/did"
	"github.com/web3-storage/go-ucanto/principal"
	"github.com/web3-storage/go-w3up/capability/storeadd"
	"github.com/web3-storage/go-w3up/capability/uploadadd"
	"github.com/web3-storage/go-w3up/capability/uploadlist"
)

func StoreAdd(signer principal.Signer, space did.DID, params *storeadd.Caveat, options ...Option) (receipt.Receipt[*storeadd.Success, *storeadd.Failure], error) {
	cfg := ClientConfig{conn: DefaultConnection}
	for _, opt := range options {
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}

	inv, err := invocation.Invoke(
		signer,
		cfg.conn.ID(),
		storeadd.NewCapability(space, params),
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

	reader, err := storeadd.NewReceiptReader()
	if err != nil {
		return nil, err
	}

	return reader.Read(rcptlnk, resp.Blocks())
}

func UploadAdd(signer principal.Signer, space did.DID, params *uploadadd.Caveat, options ...Option) (receipt.Receipt[*uploadadd.Success, *uploadadd.Failure], error) {
	cfg := ClientConfig{conn: DefaultConnection}
	for _, opt := range options {
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}

	inv, err := invocation.Invoke(
		signer,
		cfg.conn.ID(),
		uploadadd.NewCapability(space, params),
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

	reader, err := uploadadd.NewReceiptReader()
	if err != nil {
		return nil, err
	}

	return reader.Read(rcptlnk, resp.Blocks())
}

func UploadList(signer principal.Signer, space did.DID, params *uploadlist.Caveat, options ...Option) (receipt.Receipt[*uploadlist.Success, *uploadlist.Failure], error) {
	cfg := ClientConfig{conn: DefaultConnection}
	for _, opt := range options {
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}

	inv, err := invocation.Invoke(
		signer,
		cfg.conn.ID(),
		uploadlist.NewCapability(space, params),
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
