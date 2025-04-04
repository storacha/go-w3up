# go-w3up

A w3up client in golang. ⚠️ Heavily WIP.

## Install

```sh
go get github.com/storacha/go-w3up
```

## Usage

⚠️ Heavily WIP. At time of writing the client/CLI does not store delegations or select matching delegations for invocations. It is necessary to provide pre-selected proofs (aka delegations) when making invocations. The easiest way to obtain proofs is to use the w3up JS CLI in your local environment and delegate capabilities to the DID you'd like to use in golang. Check the [how to for obtaining proofs](#obtain-proofs).

### Client library

To use the client library, you should first [generate a DID](#generate-a-did) and then delegate capabilities allowing the generated DID to perform tasks. You can then use those delegations as your proofs. See the [how to for obtaining proofs](#obtain-proofs).

Example listing uploads:

```go
package main

import (
  "net/url"
  "ioutil"

  "github.com/web3-storage/go-ucanto/did"
  "github.com/web3-storage/go-ucanto/principal/ed25519/signer"
  "github.com/storacha/go-w3up/client"
  "github.com/storacha/go-w3up/delegation"
)

func main(

	// private key to sign invocation UCAN with
	priv,_:= ioutil.ReadFile("path/to/private.key")
	signer, _ := signer.Parse(priv)

	// UCAN proof that signer can list uploads in this space (a delegation chain)
	prfbytes, _ := ioutil.ReadFile("path/to/proof.ucan")
	proof, _ := delegation.ExtractProof(b)

	// space to list uploads from
	space, _ := did.Parse("did:key:z6MkwDuRThQcyWjqNsK54yKAmzfsiH6BTkASyiucThMtHt1y")

	rcpt, _ := client.UploadList(
	signer,
	space,
	&uploadlist.Caveat{},
	client.WithProof(proof),
	)

	for _, r := range rcpt.Out().Ok().Results {
	fmt.Printf("%s\n", r.Root)
	}

)
```

### CLI

The CLI will automatically generate a DID for you and store it in `~/.w3up/config`. To use the CLI, you should delegate capabilities allowing that DID to perform tasks. You can then use those delegations as your proofs. You can use `go run ./cmd/w3 whoami` to print the DID (public key) - this is the DID you should delegate capabilities to. See the [how to for obtaining proofs](#obtain-proofs), optionally skipping the first step since the CLI already generated a DID for you.

```console
go run ./cmd/w3.go --help
NAME:
   w3 - interact with the web3.storage API

USAGE:
   w3 [global options] command [command options] [arguments...]

COMMANDS:
   whoami      Print information about the current agent.
   up, upload  Store a file(s) to the service and register an upload.
   ls, list    List uploads in the current space.
   help, h     Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h  show help
```

## How to

### Generate a DID

You can use `ucan-key` to generate a private key and DID for use with the library. Install Node.js and then use the `ucan-key` module:

```sh
npx ucan-key ed
```

Output should look something like:

```sh
# did:key:z6Mkh9TtUbFJcUHhMmS9dEbqpBbHPbL9oxg1zziWn1CYCNZ2
MgCb+bRGl02JqlWMPUxCyntxlYj0T/zLtR2tn8LFvw6+Yke0BKAP/OUu2tXpd+tniEoOzB3pxqxHZpRhrZl1UYUeraT0=
```
You can use the private key (the line starting `Mg...`) in the CLI by setting the environment variable `W3UP_PRIVATE_KEY`. Alternatively you can use it programmatically after parsing it:

```go
package main

import "github.com/web3-storage/go-ucanto/principal/ed25519/signer"

signer, _ := signer.Parse("MgCb+bRGl02JqlWMPUxCyntxlYj0T/zLtR2tn8LFvw6+Yke0BKAP/OUu2tXpd+tniEoOzB3pxqxHZpRhrZl1UYUeraT0=")
```

### Obtain proofs

Proofs are delegations to your DID enabling it to perform tasks. Currently the best way to obtain proofs that will allow you to interact with the web3.storage API is to use the w3up JS CLI:

1. [Generate a DID](#generate-a-did) and make a note of it (the string starting with `did:key:...`)
1. Install w3 CLI:
    ```sh
    npm install -g @web3-storage/w3cli
    ```
1. Create a space:
    ```sh
    w3 space create <NAME>
    ```
1. Delegate capabilities to your DID:
    ```sh
    w3 delegation create -c 'store/*' -c 'upload/*' <DID>`
    ```

## API

[pkg.go.dev Reference](https://pkg.go.dev/github.com/storacha/go-w3up)

## Contributing

Feel free to join in. All welcome. Please [open an issue](https://github.com/storacha/go-w3up/issues)!

## License

Dual-licensed under [MIT + Apache 2.0](LICENSE.md)
