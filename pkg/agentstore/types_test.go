package agentstore

import (
	"crypto/rand"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/fil-forge/libforge/commands/blob"
	"github.com/fil-forge/ucantone/did"
	"github.com/fil-forge/ucantone/testutil"
	"github.com/fil-forge/ucantone/ucan"
	"github.com/fil-forge/ucantone/ucan/delegation"
	"github.com/fil-forge/ucantone/ucan/delegation/policy"
	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
	"github.com/stretchr/testify/require"
)

func TestRoundTripAgentData(t *testing.T) {
	agentPrincipal := testutil.RandomSigner(t)

	del, err := newDelegation(t)

	require.NoError(t, err)

	agentData := AgentData{
		Principal:   agentPrincipal,
		Delegations: []ucan.Delegation{del},
	}

	str, err := json.Marshal(agentData)
	require.NoError(t, err)

	var agentDataReturned AgentData
	err = json.Unmarshal(str, &agentDataReturned)
	require.NoError(t, err)

	require.Equal(t, agentData.Principal, agentDataReturned.Principal)
	require.Equal(t, delegationsCIDs(agentData), delegationsCIDs(agentDataReturned))
}

func TestWriteReadAgentData(t *testing.T) {
	dataFilePath := filepath.Join(t.TempDir(), "agentdata.json")

	agentPrincipal := testutil.RandomSigner(t)
	del, err := newDelegation(t)
	require.NoError(t, err)

	agentData := AgentData{
		Principal:   agentPrincipal,
		Delegations: []ucan.Delegation{del},
	}

	err = writeToFile(dataFilePath, agentData)
	require.NoError(t, err)

	agentDataReturned, err := readFromFile(dataFilePath)
	require.NoError(t, err)

	require.Equal(t, agentData.Principal, agentDataReturned.Principal)
	require.Equal(t, delegationsCIDs(agentData), delegationsCIDs(agentDataReturned))
}

func newDelegation(t *testing.T) (ucan.Delegation, error) {
	t.Helper()

	signer := testutil.RandomSigner(t)

	audienceDid, err := did.Parse("did:mailto:example.com:alice")
	if err != nil {
		return nil, err
	}

	bytes := make([]byte, 128)
	_, err = rand.Read(bytes)
	if err != nil {
		return nil, err
	}

	digest, err := multihash.Sum(bytes, multihash.SHA2_256, -1)
	if err != nil {
		return nil, err
	}

	return blob.Add.Delegate(
		signer,
		audienceDid,
		signer.DID(),
		delegation.WithPolicyBuilder(
			policy.Equal(".blob", blob.Blob{
				Digest: digest,
				Size:   uint64(len(bytes)),
			}),
		),
	)
}

func delegationsCIDs(d AgentData) []cid.Cid {
	cids := make([]cid.Cid, len(d.Delegations))
	for i, d := range d.Delegations {
		cids[i] = d.Link()
	}
	return cids
}
