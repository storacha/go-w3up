package client

import (
	"bytes"
	"context"
	"fmt"

	logging "github.com/ipfs/go-log/v2"
	cbg "github.com/whyrusleeping/cbor-gen"
	"go.opentelemetry.io/otel"

	"github.com/fil-forge/ucantone/did"
	"github.com/fil-forge/ucantone/errors/datamodel"
	"github.com/fil-forge/ucantone/execution"
	"github.com/fil-forge/ucantone/ucan"
	"github.com/fil-forge/ucantone/ucan/invocation"
	"github.com/fil-forge/ucantone/validator/bindcom"

	"github.com/storacha/guppy/pkg/agentstore"
	// receiptclient "github.com/storacha/guppy/pkg/receipt"
)

var (
	log    = logging.Logger("github.com/storacha/guppy/pkg/client")
	tracer = otel.Tracer("github.com/storacha/guppy/pkg/client")
)

type Client struct {
	ucantoneClient UcantoneClient
	// receiptsClient   *receiptclient.Client
	service          did.DID
	store            agentstore.Store
	additionalProofs []ucan.Delegation
	// retrievalOpts    []rclient.Option
}

// NewClient creates a new client.
func NewClient(options ...Option) (*Client, error) {
	c := Client{
		ucantoneClient: DefaultUcantoneClient,
		// receiptsClient: DefaultReceiptsClient,
	}

	for _, opt := range options {
		if err := opt(&c); err != nil {
			return nil, err
		}
	}

	// Create a default memory store if none provided
	if c.store == nil {
		store, err := agentstore.NewMemory()
		if err != nil {
			return nil, fmt.Errorf("creating default memory store: %w", err)
		}
		c.store = store
	}

	return &c, nil
}

// DID returns the DID of the client.
func (c *Client) DID() did.DID {
	p, err := c.store.Principal()
	if err != nil {
		log.Warnf("getting principal: %s", err)
		return did.DID{}
	}
	if p == nil {
		return did.DID{}
	}
	return p.DID()
}

// UcantoneClient returns the underlying Ucantone client used by the client.
func (c *Client) UcantoneClient() UcantoneClient {
	return c.ucantoneClient
}

// Issuer returns the issuing signer of the client.
func (c *Client) Issuer() ucan.Signer {
	p, err := c.store.Principal()
	if err != nil {
		log.Warnf("getting principal: %s", err)
		return nil
	}
	return p
}

// Proofs returns delegations that match the given capability queries.
// If no queries are provided, returns all non-expired delegations.
// Delegations are filtered by:
//   - Expiration: excludes expired delegations
//   - NotBefore: excludes delegations that are not yet valid
//   - Capability matching: if queries are provided, only returns delegations
//     whose capabilities match at least one of the queries
//
// Additionally, this method includes relevant session proofs (ucan/attest delegations)
// that attest to the returned authorizations.
//
// Returns both stored delegations (from store) and additional proofs
// (from c.additionalProofs).
func (c *Client) Proofs(queries ...agentstore.CapabilityQuery) ([]ucan.Delegation, error) {
	// Get delegations from store
	storeDelegations, err := c.store.Query(queries...)
	if err != nil {
		return nil, fmt.Errorf("querying delegations: %w", err)
	}

	// If no additional proofs, return store results directly
	if len(c.additionalProofs) == 0 {
		return storeDelegations, nil
	}

	// Query additional proofs using the same query logic
	additionalResults := agentstore.Query(c.additionalProofs, queries)

	// Combine results, deduplicating by CID
	seen := make(map[string]struct{})
	res := make([]ucan.Delegation, 0, len(storeDelegations)+len(additionalResults))

	for _, del := range storeDelegations {
		cidStr := del.Link().String()
		if _, exists := seen[cidStr]; !exists {
			seen[cidStr] = struct{}{}
			res = append(res, del)
		}
	}

	for _, del := range additionalResults {
		cidStr := del.Link().String()
		if _, exists := seen[cidStr]; !exists {
			seen[cidStr] = struct{}{}
			res = append(res, del)
		}
	}

	return res, nil
}

// AddProofs adds the given delegations to the client's store.
func (c *Client) AddProofs(delegations ...ucan.Delegation) error {
	return c.store.AddDelegations(delegations...)
}

// Reset clears all delegations from the store while preserving the principal.
func (c *Client) Reset() error {
	return c.store.Reset()
}

func invokeAndExecute[A bindcom.Arguments](
	ctx context.Context,
	c *Client,
	subject did.DID,
	command bindcom.Command[A],
	arguments A,
	options ...invocation.Option,
) (execution.Response, error) {
	inv, err := command.Invoke(c.Issuer(), subject, arguments, options...)
	if err != nil {
		return nil, err
	}

	return c.ucantoneClient.Execute(
		execution.NewRequest(ctx, inv),
	)
}

// unpack takes an execution response and attempts to unpack the "ok" value into
// the provided CBOR unmarshaler. If the response contains an "error" instead,
// it attempts to unpack that into a datamodel.ErrorModel and return it as an
// error. If the "ok" or "error" value cannot be unpacked, it returns that
// error.
func unpack(res execution.Response, ok cbg.CBORUnmarshaler) error {
	okBytes, errBytes := res.Receipt().Out().Unpack()
	if errBytes != nil {
		var err datamodel.ErrorModel
		err.UnmarshalCBOR(bytes.NewReader(errBytes))
		return err
	}

	err := ok.UnmarshalCBOR(bytes.NewReader(okBytes))
	if err != nil {
		return err
	}

	return nil
}

func invokeExecuteAndUnpack[A bindcom.Arguments](
	ctx context.Context,
	c *Client,
	subject did.DID,
	command bindcom.Command[A],
	arguments A,
	ok cbg.CBORUnmarshaler,
	options ...invocation.Option,
) error {
	res, err := invokeAndExecute(ctx, c, subject, command, arguments, options...)
	if err != nil {
		return err
	}

	err = unpack(res, ok)
	return err
}
