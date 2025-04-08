package client

import (
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"slices"

	"github.com/ipfs/go-cid"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/multiformats/go-multihash"
	"github.com/storacha/go-libstoracha/capabilities/blob"
	httpcap "github.com/storacha/go-libstoracha/capabilities/http"
	spaceblob "github.com/storacha/go-libstoracha/capabilities/space/blob"
	captypes "github.com/storacha/go-libstoracha/capabilities/types"
	ucancap "github.com/storacha/go-libstoracha/capabilities/ucan"
	"github.com/storacha/go-ucanto/client"
	"github.com/storacha/go-ucanto/core/dag/blockstore"
	"github.com/storacha/go-ucanto/core/delegation"
	"github.com/storacha/go-ucanto/core/invocation"
	"github.com/storacha/go-ucanto/core/invocation/ran"
	"github.com/storacha/go-ucanto/core/ipld"
	"github.com/storacha/go-ucanto/core/receipt"
	"github.com/storacha/go-ucanto/core/result"
	"github.com/storacha/go-ucanto/core/result/failure"
	fdm "github.com/storacha/go-ucanto/core/result/failure/datamodel"
	"github.com/storacha/go-ucanto/did"
	"github.com/storacha/go-ucanto/principal"
	"github.com/storacha/go-ucanto/principal/ed25519/signer"
	"github.com/storacha/go-ucanto/ucan"
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
func StoreAdd(issuer principal.Signer, space did.DID, params storeadd.Caveat, options ...Option) (receipt.Receipt[*storeadd.Success, *storeadd.Failure], error) {
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
func UploadAdd(issuer principal.Signer, space did.DID, params uploadadd.Caveat, options ...Option) (receipt.Receipt[*uploadadd.Success, *uploadadd.Failure], error) {
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
func UploadList(issuer principal.Signer, space did.DID, params uploadlist.Caveat, options ...Option) (receipt.Receipt[*uploadlist.Success, *uploadlist.Failure], error) {
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

// BlobAdd adds a blob to the service. The issuer needs proof of
// `space/blob/add` delegated capability.
//
// Required delegated capability proofs: `space/blob/add`
//
// The `issuer` is the signing authority that is issuing the UCAN invocation.
//
// The `space` is the resource the invocation applies to. It is typically the
// DID of a space.
func BlobAdd(content io.Reader, issuer principal.Signer, space did.DID, options ...Option) (ipld.Link, delegation.Delegation, error) {
	cfg := ClientConfig{conn: DefaultConnection}
	for _, opt := range options {
		if err := opt(&cfg); err != nil {
			return nil, nil, err
		}
	}

	contentBytes, err := io.ReadAll(content)
	if err != nil {
		return nil, nil, fmt.Errorf("reading content: %w", err)
	}

	contentHash, err := multihash.Sum(contentBytes, multihash.SHA2_256, -1)
	if err != nil {
		return nil, nil, fmt.Errorf("computing content multihash: %w", err)
	}

	contentLink := cidlink.Link{Cid: cid.NewCidV1(0x0202, contentHash)}

	caveats := spaceblob.AddCaveats{
		Blob: spaceblob.Blob{
			Digest: contentHash,
			Size:   uint64(len(contentBytes)),
		},
	}

	inv, err := spaceblob.Add.Invoke(issuer, cfg.conn.ID(), space.String(), caveats, convertToInvocationOptions(cfg)...)
	if err != nil {
		return nil, nil, fmt.Errorf("generating invocation: %w", err)
	}

	resp, err := client.Execute([]invocation.Invocation{inv}, cfg.conn)
	if err != nil {
		return nil, nil, fmt.Errorf("sending invocation: %w", err)
	}

	rcptlnk, ok := resp.Get(inv.Link())
	if !ok {
		return nil, nil, fmt.Errorf("receipt not found: %s", inv.Link())
	}

	reader, err := receipt.NewReceiptReaderFromTypes[spaceblob.AddOk, fdm.FailureModel](spaceblob.AddOkType(), fdm.FailureType())
	if err != nil {
		return nil, nil, fmt.Errorf("generating receipt reader: %w", err)
	}

	rcpt, err := reader.Read(rcptlnk, resp.Blocks())
	if err != nil {
		return nil, nil, fmt.Errorf("reading receipt: %w", err)
	}

	_, err = result.Unwrap(result.MapError(rcpt.Out(), failure.FromFailureModel))
	if err != nil {
		return nil, nil, fmt.Errorf("blob add failed: %w", err)
	}

	var allocateTask, putTask, acceptTask invocation.Invocation
	var concludeFxs []invocation.Invocation
	for _, task := range rcpt.Fx().Fork() {
		inv, ok := task.Invocation()
		if ok {
			switch inv.Capabilities()[0].Can() {
			case blob.AllocateAbility:
				allocateTask = inv
			case ucancap.ConcludeAbility:
				concludeFxs = append(concludeFxs, inv)
			case httpcap.PutAbility:
				putTask = inv
			case blob.AcceptAbility:
				acceptTask = inv
			}
		}
	}

	if allocateTask == nil || len(concludeFxs) == 0 || putTask == nil || acceptTask == nil {
		return nil, nil, fmt.Errorf("mandatory tasks not received in space/blob/add receipt")
	}

	var allocateRcpt receipt.Receipt[blob.AllocateOk, fdm.FailureModel]
	var putRcpt receipt.AnyReceipt
	var acceptRcpt receipt.Receipt[*blob.AcceptOk, fdm.FailureModel]
	for _, concludeFx := range concludeFxs {
		concludeRcpt, err := getConcludeReceipt(concludeFx)
		if err != nil {
			return nil, nil, fmt.Errorf("reading ucan/conclude receipt: %w", err)
		}

		switch concludeRcpt.Ran().Link() {
		case allocateTask.Link():
			allocateRcpt, err = receipt.Rebind[blob.AllocateOk, fdm.FailureModel](concludeRcpt, blob.AllocateOkType(), fdm.FailureType(), captypes.Converters...)
			if err != nil {
				return nil, nil, fmt.Errorf("bad allocate receipt in conclude fx: %w", err)
			}
		case putTask.Link():
			putRcpt = concludeRcpt
		case acceptTask.Link():
			acceptRcpt, err = receipt.Rebind[*blob.AcceptOk, fdm.FailureModel](concludeRcpt, blob.AcceptOkType(), fdm.FailureType(), captypes.Converters...)
			if err != nil {
				return nil, nil, fmt.Errorf("bad accept receipt in conclude fx: %w", err)
			}
		}
	}

	if allocateRcpt == nil {
		return nil, nil, fmt.Errorf("mandatory receipts not received in space/blob/add receipt")
	}

	allocateOk, err := result.Unwrap(result.MapError(allocateRcpt.Out(), failure.FromFailureModel))
	if err != nil {
		return nil, nil, fmt.Errorf("blob allocation failed: %w", err)
	}

	// put the blob where assigned
	address := allocateOk.Address
	if address == nil {
		return nil, nil, fmt.Errorf("blob allocation failed: no address")
	}

	if err := putBlob(address.URL, address.Headers, content); err != nil {
		return nil, nil, fmt.Errorf("putting blob: %w", err)
	}

	// invoke `ucan/conclude` with `http/put` receipt
	if putRcpt != nil {
		putOk, _ := result.Unwrap(putRcpt.Out())
		if putOk == nil {
			putKeys, ok := putTask.Facts()[0]["keys"].(map[string]any)
			if !ok {
				return nil, nil, fmt.Errorf("invalid put facts")
			}

			derivedKey, ok := slices.Collect(maps.Values(putKeys))[0].([]byte)
			if !ok {
				return nil, nil, fmt.Errorf("invalid derived key")
			}

			derivedSigner, err := signer.FromRaw(derivedKey)
			if err != nil {
				return nil, nil, fmt.Errorf("deriving signer: %w", err)
			}

			putRcpt, err = receipt.Issue(derivedSigner, result.Ok[httpcap.PutOk, ipld.Builder](httpcap.PutOk{}), ran.FromLink(putTask.Link()))
			if err != nil {
				return nil, nil, fmt.Errorf("generating receipt: %w", err)
			}

			// var concludeFacts []ucan.FactBuilder
			// for rcptBlock, err := range putRcpt.Blocks() {
			// 	if err != nil {
			// 		return nil, nil, fmt.Errorf("getting receipt block: %w", err)
			// 	}

			// 	concludeFacts = append(concludeFacts, rcptBlock.Link())
			// }

			httpPutConcludeInvocation, err := ucancap.Conclude.Invoke(
				derivedSigner,
				cfg.conn.ID(),
				issuer.DID().String(),
				ucancap.ConcludeCaveats{
					Receipt: putRcpt.Root().Link(),
				},
				// delegation.WithFacts(concludeFacts),
				delegation.WithNoExpiration(),
			)
			if err != nil {
				return nil, nil, fmt.Errorf("generating invocation: %w", err)
			}

			// attach the receipt to the conclude invocation
			for rcptBlock, err := range putRcpt.Blocks() {
				if err != nil {
					return nil, nil, fmt.Errorf("getting receipt block: %w", err)
				}
				httpPutConcludeInvocation.Attach(rcptBlock)
			}

			resp, err := client.Execute([]invocation.Invocation{httpPutConcludeInvocation}, cfg.conn)
			if err != nil {
				return nil, nil, fmt.Errorf("executing conclude invocation: %w", err)
			}

			rcptlnk, ok := resp.Get(httpPutConcludeInvocation.Link())
			if !ok {
				return nil, nil, fmt.Errorf("receipt not found: %s", httpPutConcludeInvocation.Link())
			}

			reader, err := receipt.NewReceiptReaderFromTypes[ucancap.ConcludeOk, fdm.FailureModel](ucancap.ConcludeOkType(), fdm.FailureType(), captypes.Converters...)
			if err != nil {
				return nil, nil, fmt.Errorf("generating receipt reader: %w", err)
			}

			rcpt, err := reader.Read(rcptlnk, resp.Blocks())
			if err != nil {
				return nil, nil, fmt.Errorf("reading receipt: %w", err)
			}

			_, err = result.Unwrap(result.MapError(rcpt.Out(), failure.FromFailureModel))
			if err != nil {
				return nil, nil, fmt.Errorf("ucan/conclude failed: %w", err)
			}
		}
	}

	// ensure the blob has been accepted
	if acceptRcpt != nil {
		acceptOk, _ := result.Unwrap(acceptRcpt.Out())
		if acceptOk == nil {
			acceptRcpt, err = pollAccept(acceptTask.Link(), cfg.conn)
			if err != nil {
				return nil, nil, fmt.Errorf("polling blob accept: %w", err)
			}

			_, acceptFail := result.Unwrap(result.MapError(acceptRcpt.Out(), failure.FromFailureModel))
			if acceptFail != nil {
				return nil, nil, fmt.Errorf("blob/accept failed: %w", acceptFail)
			}
		}
	}

	acceptOk, _ := result.Unwrap(acceptRcpt.Out())

	locationBlocks, err := blockstore.NewBlockStore(blockstore.WithBlocksIterator(acceptRcpt.Blocks()))
	if err != nil {
		return nil, nil, fmt.Errorf("reading location commitment blocks: %w", err)
	}

	location, err := delegation.NewDelegationView(acceptOk.Site, locationBlocks)
	if err != nil {
		return nil, nil, fmt.Errorf("creating location delegation: %w", err)
	}

	return contentLink, location, nil
}

func getConcludeReceipt(concludeFx invocation.Invocation) (receipt.AnyReceipt, error) {
	concludeNb, fail := ucancap.ConcludeCaveatsReader.Read(concludeFx.Capabilities()[0].Nb())
	if fail != nil {
		return nil, fmt.Errorf("invalid conclude receipt: %w", fail)
	}

	reader := receipt.NewAnyReceiptReader(captypes.Converters...)
	rcpt, err := reader.Read(concludeNb.Receipt, concludeFx.Blocks())
	if err != nil {
		return nil, fmt.Errorf("reading receipt: %w", err)
	}

	return rcpt, nil
}

func putBlob(url url.URL, headers http.Header, body io.Reader) error {
	req, err := http.NewRequest(http.MethodPut, url.String(), body)
	if err != nil {
		return fmt.Errorf("creating upload request: %w", err)
	}

	for k, v := range headers {
		req.Header.Set(k, v[0])
	}

	// TODO: custom HTTP client with timeout
	// TODO: retries
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("uploading blob: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("uploading blob: %s", resp.Status)
	}

	return nil
}

func pollAccept(link ucan.Link, conn client.Connection) (receipt.Receipt[*blob.AcceptOk, fdm.FailureModel], error) {
	rcpt, err := receipt.Issue(ucan.Signer(signer.Ed25519Signer{}), result.Ok[*blob.AcceptOk, ipld.Builder](&blob.AcceptOk{}), ran.FromLink(link))
	if err != nil {
		return nil, fmt.Errorf("generating receipt: %w", err)
	}

	return receipt.Rebind[*blob.AcceptOk, fdm.FailureModel](rcpt, blob.AcceptOkType(), fdm.FailureType())
}
