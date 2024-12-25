package cmd

import (
	"context"
	"fmt"
	"wrappedeal/internal/filecoin"
	"wrappedeal/internal/utils"

	"github.com/urfave/cli/v2"
)

var FilCmd = &cli.Command{
	Name:    "fil",
	Aliases: []string{"f"},
	Usage:   "Filecoin related commands",
	Subcommands: []*cli.Command{
		{
			Name:  "deal",
			Usage: "Make an online deal with wrappedeal",
			Flags: append([]cli.Flag{
				&cli.StringFlag{
					Name:     "http-url",
					Usage:    "http url to CAR file",
					Required: true,
				},
				&cli.StringSliceFlag{
					Name:  "http-headers",
					Usage: "http headers to be passed with the request (e.g key=value)",
				},
				&cli.Uint64Flag{
					Name:     "car-size",
					Usage:    "size of the CAR file: required for online deals",
					Required: true,
				},
			}, dealFlags...),
			Action: func(cctx *cli.Context) error {
				return filecoin.DealCmdAction(cctx, true)
			},
		},
		{
			Name:  "local-deal",
			Usage: "Make deal from any local file/folder with wrappedeal",
			Flags: localDealFlags,
			Action: func(cctx *cli.Context) error {
				return filecoin.LocalDealCmdAction(cctx, true)

			},
		},
		{
			Name:  "offline-deal",
			Usage: "Make an offline deal with wrappedeal",
			Flags: dealFlags,
			Action: func(cctx *cli.Context) error {
				return filecoin.DealCmdAction(cctx, false)
			},
		},
		{
			Name:    "get-eth-addr",
			Aliases: []string{"gea"},
			Usage:   "Get the Ethereum address corresponding to a Filecoin address",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "filecoin-addr",
					Aliases:  []string{"f"},
					Usage:    "Filecoin address to convert",
					Required: true,
				},
				&cli.StringFlag{
					Name:     "repo",
					Aliases:  []string{"R"},
					Usage:    "Boost client repository directory path (default: ~/.boost-client)",
					Value:    "~/.boost-client",
					Required: false,
				},
			},
			Action: func(c *cli.Context) error {
				ctx := context.Background()
				filecoinAddrStr := c.String("filecoin-addr")
				repoFlag := c.String("repo")

				// Expand the repo path
				repoPath, err := utils.ExpandPath(repoFlag)
				if err != nil {
					return fmt.Errorf("failed to expand repo path: %v", err)
				}
				return filecoin.GetEthAddr(ctx, filecoinAddrStr, repoPath)
			},
		},
	},
}

var dealFlags = []cli.Flag{
	&cli.StringFlag{
		Name:     "provider",
		Usage:    "storage provider on-chain address",
		Required: true,
	},
	&cli.StringFlag{
		Name:     "commp",
		Usage:    "commp of the CAR file",
		Required: true,
	},
	&cli.Uint64Flag{
		Name:     "piece-size",
		Usage:    "size of the CAR file as a padded piece",
		Required: true,
	},
	&cli.StringFlag{
		Name:     "payload-cid",
		Usage:    "root CID of the CAR file",
		Required: true,
	},
	&cli.IntFlag{
		Name:  "start-epoch-head-offset",
		Usage: "start epoch by when the deal should be proved by provider on-chain after current chain head",
	},
	&cli.IntFlag{
		Name:  "start-epoch",
		Usage: "start epoch by when the deal should be proved by provider on-chain",
	},
	&cli.IntFlag{
		Name:  "duration",
		Usage: "duration of the deal in epochs",
		Value: 518400, // default is 2880 * 180 == 180 days
	},
	&cli.IntFlag{
		Name:  "provider-collateral",
		Usage: "deal collateral that storage miner must put in escrow; if empty, the min collateral for the given piece size will be used",
	},
	&cli.Int64Flag{
		Name:  "storage-price",
		Usage: "storage price in attoFIL per epoch per GiB",
		Value: 1,
	},
	&cli.BoolFlag{
		Name:  "verified",
		Usage: "whether the deal funds should come from verified client data-cap",
		Value: true,
	},
	&cli.BoolFlag{
		Name:  "remove-unsealed-copy",
		Usage: "indicates that an unsealed copy of the sector in not required for fast retrieval",
		Value: false,
	},
	&cli.StringFlag{
		Name:  "wallet",
		Usage: "filecoin wallet address whitelisted to make deals in wrapedeal contract",
	},
	&cli.BoolFlag{
		Name:  "skip-ipni-announce",
		Usage: "indicates that deal index should not be announced to the IPNI(Network Indexer)",
		Value: false,
	},
	&cli.StringFlag{
		Name:     "repo",
		Aliases:  []string{"R"},
		Usage:    "Boost client repository directory path (default: ~/.boost-client)",
		Value:    "~/.boost-client",
		Required: false,
	},
	&cli.StringFlag{
		Name:     "contract",
		Usage:    "contract address to make deal with",
		Aliases:  []string{"c"},
		Required: true,
	},
}

var localDealFlags = []cli.Flag{
	&cli.StringFlag{
		Name:     "path",
		Usage:    "Path to the file or folder to make the deal",
		Required: true,
	},
	&cli.BoolFlag{
		Name:  "lighthouse",
		Usage: "Use Lighthouse as a buffer",
		Value: false,
	},
	&cli.StringFlag{
		Name:  "apikey",
		Usage: "API key for Lighthouse (overrides .env)",
	},
	&cli.StringFlag{
		Name:     "provider",
		Usage:    "storage provider on-chain address",
		Required: true,
	},
	&cli.StringFlag{
		Name:  "payload-cid",
		Usage: "root CID of the CAR file",
	},
	&cli.IntFlag{
		Name:  "start-epoch-head-offset",
		Usage: "start epoch by when the deal should be proved by provider on-chain after current chain head",
	},
	&cli.IntFlag{
		Name:  "start-epoch",
		Usage: "start epoch by when the deal should be proved by provider on-chain",
	},
	&cli.IntFlag{
		Name:  "duration",
		Usage: "duration of the deal in epochs",
		Value: 518400, // default is 2880 * 180 == 180 days
	},
	&cli.IntFlag{
		Name:  "provider-collateral",
		Usage: "deal collateral that storage miner must put in escrow; if empty, the min collateral for the given piece size will be used",
	},
	&cli.Int64Flag{
		Name:  "storage-price",
		Usage: "storage price in attoFIL per epoch per GiB",
		Value: 1,
	},
	&cli.BoolFlag{
		Name:  "verified",
		Usage: "whether the deal funds should come from verified client data-cap",
		Value: true,
	},
	&cli.BoolFlag{
		Name:  "remove-unsealed-copy",
		Usage: "indicates that an unsealed copy of the sector in not required for fast retrieval",
		Value: false,
	},
	&cli.StringFlag{
		Name:  "wallet",
		Usage: "wallet address to be used to initiate the deal",
	},
	&cli.BoolFlag{
		Name:  "skip-ipni-announce",
		Usage: "indicates that deal index should not be announced to the IPNI(Network Indexer)",
		Value: false,
	},
	&cli.StringFlag{
		Name:  "http-url",
		Usage: "http url to CAR file",
	},
	&cli.StringSliceFlag{
		Name:  "http-headers",
		Usage: "http headers to be passed with the request (e.g key=value)",
	},
	&cli.Uint64Flag{
		Name:  "piece-size",
		Usage: "size of the CAR file as a padded piece",
	},
	&cli.StringFlag{
		Name:  "out-dir",
		Usage: "output directory for CAR files",
		Value: "/tmp",
	},
	&cli.StringFlag{
		Name:     "repo",
		Aliases:  []string{"R"},
		Usage:    "Boost client repository directory path (default: ~/.boost-client)",
		Value:    "~/.boost-client",
		Required: false,
	},
	&cli.StringFlag{
		Name:     "contract",
		Aliases:  []string{"c"},
		Usage:    "contract address to make deal with",
		Required: true,
	},
}
