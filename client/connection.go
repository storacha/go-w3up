package client

import (
	"log"
	"net/url"

	"github.com/web3-storage/go-ucanto/client"
	"github.com/web3-storage/go-ucanto/did"
	"github.com/web3-storage/go-ucanto/transport/car"
	"github.com/web3-storage/go-ucanto/transport/http"
)

var DefaultConnection client.Connection

func init() {
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

	DefaultConnection = conn
}
