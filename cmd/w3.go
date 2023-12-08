package main

import (
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli/v2"
	"github.com/web3-storage/go-ucanto/core/delegation"
	"github.com/web3-storage/go-w3up/capability/uploadlist"
	"github.com/web3-storage/go-w3up/client"
	"github.com/web3-storage/go-w3up/cmd/lib"
)

func main() {
	app := &cli.App{
		Name:  "w3",
		Usage: "interact with the web3.storage API",
		Commands: []*cli.Command{
			{
				Name:   "whoami",
				Usage:  "Print information about the current agent.",
				Action: whoami,
			},
			{
				Name:    "ls",
				Aliases: []string{"list"},
				Usage:   "List uploads in the current space.",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "space",
						Value: "",
						Usage: "DID of space to list uploads from.",
					},
					&cli.StringFlag{
						Name:  "proof",
						Value: "",
						Usage: "Path to file containing UCAN proof(s) for the operation.",
					},
					&cli.BoolFlag{
						Name:  "shards",
						Value: false,
						Usage: "Display shard CID(s) for each upload root.",
					},
				},
				Action: ls,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func whoami(cCtx *cli.Context) error {
	s := lib.MustGetSigner()
	fmt.Println(s.DID())
	return nil
}

func ls(cCtx *cli.Context) error {
	signer := lib.MustGetSigner()
	conn := lib.MustGetConnection()
	space := lib.MustParseDID(cCtx.String("space"))
	proof := lib.MustGetProof(cCtx.String("proof"))

	rcpt, err := client.List(
		signer,
		space,
		&uploadlist.Caveat{},
		client.WithConnection(conn),
		client.WithProofs([]delegation.Delegation{proof}),
	)
	if err != nil {
		return err
	}

	if rcpt.Out().Error() != nil {
		log.Fatalf("%+v\n", rcpt.Out().Error())
	}

	for _, r := range rcpt.Out().Ok().Results {
		fmt.Printf("%s\n", r.Root)
		if cCtx.Bool("shards") {
			for _, s := range r.Shards {
				fmt.Printf("\t%s\n", s)
			}
		}
	}

	return nil
}
