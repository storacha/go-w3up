package util

import (
	_ "embed"
	"log"
	"net/url"
	"os"
	"path"

	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/codec/dagcbor"
	"github.com/ipld/go-ipld-prime/schema"
	"github.com/storacha/go-ucanto/client"
	"github.com/storacha/go-ucanto/core/delegation"
	"github.com/storacha/go-ucanto/did"
	"github.com/storacha/go-ucanto/principal"
	"github.com/storacha/go-ucanto/principal/ed25519/signer"
	"github.com/storacha/go-ucanto/transport/car"
	"github.com/storacha/go-ucanto/transport/http"
	cdg "github.com/storacha/go-w3up/delegation"
)

//go:embed config.ipldsch
var configsch []byte

type configurationModel struct {
	Signer []byte
}

func MustGetSigner() principal.Signer {
	str := os.Getenv("W3UP_PRIVATE_KEY") // use env var preferably
	if str != "" {
		s, err := signer.Parse(str)
		if err != nil {
			log.Fatal(err)
		}
		return s
	}

	conf := mustReadConfig()
	s, err := signer.Decode(conf.Signer)
	if err != nil {
		log.Fatalf("decoding signer: %s", err)
	}
	return s
}

func mustLoadConfigSchema() *schema.TypeSystem {
	ts, err := ipld.LoadSchemaBytes(configsch)
	if err != nil {
		log.Fatalf("failed to load IPLD schema: %s", err)
	}
	return ts
}

func mustReadConfig() *configurationModel {
	homedir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("obtaining user home directory: %s", err)
	}

	typ := mustLoadConfigSchema().TypeByName("Configuration")
	confdir := path.Join(homedir, ".w3up")
	confpath := path.Join(confdir, "config")
	conf := configurationModel{}

	bytes, err := os.ReadFile(confpath)
	if err != nil {
		s, err := signer.Generate()
		if err != nil {
			log.Fatalf("generating signer: %s", err)
		}

		conf.Signer = s.Encode()
		bytes, err = ipld.Marshal(dagcbor.Encode, &conf, typ)
		if err != nil {
			log.Fatalf("encoding config: %s", err)
		}
		if err := os.Mkdir(confdir, 0700); err != nil {
			log.Fatalf("writing config: %s", err)
		}
		if os.WriteFile(confpath, bytes, 0600); err != nil {
			log.Fatalf("writing config: %s", err)
		}
	} else {
		_, err = ipld.Unmarshal(bytes, dagcbor.Decode, &conf, typ)
		if err != nil {
			log.Fatalf("decoding config: %s", err)
		}
	}

	return &conf
}

func MustGetConnection() client.Connection {
	// service URL & DID
	serviceURL, err := url.Parse("https://up.web3.storage")
	if err != nil {
		log.Fatal(err)
	}

	servicePrincipal, err := did.Parse("did:web:web3.storage")
	if err != nil {
		log.Fatal(err)
	}

	// HTTP transport and CAR encoding
	channel := http.NewHTTPChannel(serviceURL)
	codec := car.NewCAROutboundCodec()

	conn, err := client.NewConnection(servicePrincipal, codec, channel)
	if err != nil {
		log.Fatal(err)
	}

	return conn
}

func MustParseDID(str string) did.DID {
	did, err := did.Parse(str)
	if err != nil {
		log.Fatalf("parsing DID: %s", err)
	}
	return did
}

func MustGetProof(path string) delegation.Delegation {
	b, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("reading proof file: %s", err)
	}

	proof, err := cdg.ExtractProof(b)
	if err != nil {
		log.Fatal(err)
	}
	return proof
}
