package client

import (
	"fmt"

	"github.com/storacha/go-ucanto/client"
	"github.com/storacha/go-ucanto/core/invocation"
	"github.com/storacha/go-ucanto/core/receipt"
	"github.com/storacha/go-ucanto/did"
	"github.com/storacha/go-ucanto/principal"
	"github.com/storacha/go-w3up/capability/storeadd"
	"github.com/storacha/go-w3up/capability/uploadadd"
	"github.com/storacha/go-w3up/capability/uploadlist"
)

// StoreAdd stores a DAG encoded as a CAR file. The issuer needs proof of
// `store/add` delegated capability.
//
// Required delegated capability proofs: `store/add`
//
// The `issuer` is the signing authority that is issuing the UCAN invocation.
//
// The `space` is the resource the invocation applies to. It is typically the
// DID of a space.
//
// The `params` are caveats required to perform a `store/add` invocation.
func StoreAdd(issuer principal.Signer, space did.DID, params *storeadd.Caveat, options ...Option) (receipt.Receipt[*storeadd.Success, *storeadd.Failure], error) {
	cfg := ClientConfig{conn: DefaultConnection}
	for _, opt := range options {
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}

	inv, err := invocation.Invoke(
		issuer,
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
	if !ok {
		return nil, fmt.Errorf("receipt not found: %s", inv.Link())
	}

	reader, err := storeadd.NewReceiptReader()
	if err != nil {
		return nil, err
	}

	return reader.Read(rcptlnk, resp.Blocks())
}

// UploadAdd registers an "upload" with the service. The issuer needs proof of
// `upload/add` delegated capability.
//
// Required delegated capability proofs: `upload/add`
//
// The `issuer` is the signing authority that is issuing the UCAN invocation.
//
// The `space` is the resource the invocation applies to. It is typically the
// DID of a space.
//
// The `params` are caveats required to perform an `upload/add` invocation.
func UploadAdd(issuer principal.Signer, space did.DID, params *uploadadd.Caveat, options ...Option) (receipt.Receipt[*uploadadd.Success, *uploadadd.Failure], error) {
	cfg := ClientConfig{conn: DefaultConnection}
	for _, opt := range options {
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}

	inv, err := invocation.Invoke(
		issuer,
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
	if !ok {
		return nil, fmt.Errorf("receipt not found: %s", inv.Link())
	}

	reader, err := uploadadd.NewReceiptReader()
	if err != nil {
		return nil, err
	}

	return reader.Read(rcptlnk, resp.Blocks())
}

// UploadList returns a paginated list of uploads in a space.
//
// Required delegated capability proofs: `upload/list`
//
// The `issuer` is the signing authority that is issuing the UCAN invocation.
//
// The `space` is the resource the invocation applies to. It is typically the
// DID of a space.
//
// The `params` are caveats required to perform an `upload/list` invocation.
func UploadList(issuer principal.Signer, space did.DID, params *uploadlist.Caveat, options ...Option) (receipt.Receipt[*uploadlist.Success, *uploadlist.Failure], error) {
	cfg := ClientConfig{conn: DefaultConnection}
	for _, opt := range options {
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}

	inv, err := invocation.Invoke(
		issuer,
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
	if !ok {
		return nil, fmt.Errorf("receipt not found: %s", inv.Link())
	}

	reader, err := uploadlist.NewReceiptReader()
	if err != nil {
		return nil, err
	}

	return reader.Read(rcptlnk, resp.Blocks())
}
