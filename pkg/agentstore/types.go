package agentstore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multibase"
	"github.com/multiformats/go-multicodec"
	"github.com/multiformats/go-multihash"
	"github.com/multiformats/go-varint"

	"github.com/fil-forge/ucantone/did"
	"github.com/fil-forge/ucantone/principal"
	"github.com/fil-forge/ucantone/principal/ed25519"
	"github.com/fil-forge/ucantone/principal/secp256k1"
	"github.com/fil-forge/ucantone/ucan"
	"github.com/fil-forge/ucantone/ucan/delegation"
)

type Store interface {
	HasPrincipal() (bool, error)
	Principal() (principal.Signer, error)
	SetPrincipal(principal principal.Signer) error
	Delegations() ([]ucan.Delegation, error)
	AddDelegations(delegations ...ucan.Delegation) error
	Reset() error
	Query(queries ...CapabilityQuery) ([]ucan.Delegation, error)
}

// CapabilityQuery represents a query to filter proofs by capability.
type CapabilityQuery struct {
	// Cmd is the command to match (e.g., "/store/add"). Use "/" to match all commands.
	Cmd ucan.Command
	// Sub is the subject to match. Use [did.Undef] to match all subjects.
	Sub did.DID
}

type AgentData struct {
	Principal   principal.Signer
	Delegations []ucan.Delegation
}

type agentDataSerialized struct {
	Principal   []byte
	Delegations []string
}

func (ad AgentData) MarshalJSON() ([]byte, error) {
	delegations := make([]string, 0, len(ad.Delegations))
	for _, d := range ad.Delegations {
		b := d.Bytes()
		digest, err := multihash.Sum(b, uint64(multicodec.Identity), -1)
		if err != nil {
			return nil, fmt.Errorf("creating multihash: %w", err)
		}
		cid := cid.NewCidV1(uint64(multicodec.Car), digest)
		b64, err := cid.StringOfBase(multibase.Base64)
		if err != nil {
			return nil, fmt.Errorf("encoding delegation cid to base64: %w", err)
		}
		delegations = append(delegations, b64)
	}

	return json.Marshal(agentDataSerialized{
		Principal:   ad.Principal.Bytes(),
		Delegations: delegations,
	})
}

func (ad *AgentData) UnmarshalJSON(b []byte) error {
	var s agentDataSerialized
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	// Principal

	code, err := varint.ReadUvarint(bytes.NewReader(s.Principal))
	if err != nil {
		return fmt.Errorf("reading private key codec: %s", err)
	}

	switch code {
	case ed25519.Code:
		ad.Principal, err = ed25519.Decode(s.Principal)
		if err != nil {
			return err
		}

	case secp256k1.Code:
		ad.Principal, err = secp256k1.Decode(s.Principal)
		if err != nil {
			return err
		}

	default:
		return fmt.Errorf("invalid private key codec: %d", code)
	}

	// Delegations

	ad.Delegations = make([]ucan.Delegation, len(s.Delegations))
	for i, b64 := range s.Delegations {
		cid, err := cid.Decode(b64)
		if err != nil {
			return fmt.Errorf("decoding delegation cid %d: %w", i, err)
		}
		if cid.Prefix().Codec != uint64(multicodec.Car) {
			return fmt.Errorf("invalid delegation codec %d for delegation %d, expected CAR (%d)", cid.Prefix().Codec, i, multicodec.Car)
		}
		if cid.Prefix().MhType != uint64(multicodec.Identity) {
			return fmt.Errorf("invalid delegation multihash type %d for delegation %d, expected Identity (%d)", cid.Prefix().MhType, i, multicodec.Identity)
		}
		decoded, err := multihash.Decode(cid.Hash())
		if err != nil {
			return fmt.Errorf("decoding delegation multihash %d: %w", i, err)
		}
		d, err := delegation.Decode(decoded.Digest)
		if err != nil {
			return fmt.Errorf("decoding delegation %d: %w", i, err)
		}
		ad.Delegations[i] = d
	}

	return nil
}

func readFromFile(path string) (AgentData, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return AgentData{}, fmt.Errorf("failed to read agent data from %q: %w", path, err)
	}

	var ad AgentData
	err = json.Unmarshal(b, &ad)
	if err != nil {
		return AgentData{}, fmt.Errorf("failed to unmarshal agent data from %q: %w", path, err)
	}
	return ad, nil
}

func writeToFile(path string, data AgentData) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0600)
}
