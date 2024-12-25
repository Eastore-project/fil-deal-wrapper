package main

import (
	"log"
	"os"
	"wrappedeal/cmd"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "wrappedeal",
		Usage: "A CLI tool for handling Smart Contract Filecoin deals",
		Commands: []*cli.Command{
			cmd.FilCmd,
			cmd.WriteContractCmd,
			cmd.ReadContractCmd,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
