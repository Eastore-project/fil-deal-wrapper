package cmd

import (
	"context"
	"fmt"
	"strconv"
	"wrappedeal/internal/contract"
	"wrappedeal/internal/eth"
	"wrappedeal/internal/utils"

	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
)

var commonWriteFlags = []cli.Flag{
	&cli.StringFlag{
		Name:     "contract-address",
		Aliases:  []string{"c"},
		Usage:    "MarketDealWrapper contract address",
		Required: true,
	},
	&cli.StringFlag{
		Name:    "abi-path",
		Aliases: []string{"b"},
		Usage:   "Path to the MarketDealWrapper contract ABI file",
		Value:   "./contracts/out/MarketDealWrapper.sol/MarketDealWrapper.json",
	},
	&cli.StringFlag{
		Name:    "private-key",
		Aliases: []string{"k"},
		Usage:   "Private key for signing transactions (overrides .env)",
	},
	&cli.StringFlag{
		Name:    "rpc-url",
		Aliases: []string{"r"},
		Usage:   "RPC URL for the Ethereum node (overrides .env)",
	},
}

var WriteContractCmd = &cli.Command{
	Name:  "write-contract",
	Usage: "Run write functions on the MarketDealWrapper contract",
	Subcommands: []*cli.Command{
		{
			Name:    "add-sp",
			Aliases: []string{"a"},
			Usage:   "Add a storage provider to the MarketDealWrapper contract",
			Flags: append(
				commonWriteFlags,
				&cli.Uint64Flag{
					Name:     "actor-id",
					Aliases:  []string{"id"},
					Usage:    "Actor ID of the storage provider",
					Required: true,
				},
				&cli.StringFlag{
					Name:     "eth-addr",
					Aliases:  []string{"e"},
					Usage:    "Ethereum address of the storage provider to receive payments (should be 0x type f4 address)",
					Required: true,
				},
				&cli.StringFlag{
					Name:     "token",
					Aliases:  []string{"t"},
					Usage:    "ERC20 token address used for payments",
					Required: true,
				},
				&cli.Float64Flag{
					Name:     "price-per-tb-per-month",
					Aliases:  []string{"p"},
					Usage:    "Price per TB per month to be paid to Sp",
					Required: true,
				},
			),
			Action: func(c *cli.Context) error {
				ctx := context.Background()
				params := contract.StorageProviderParams{
					ActorId:              c.Uint64("actor-id"),
					EthAddr:              common.HexToAddress(c.String("eth-addr")),
					Token:                common.HexToAddress(c.String("token")),
					PricePerBytePerEpoch: utils.ConvertPrice(c.Float64("price-per-tb-per-month")),
				}

				client, err := eth.NewETHClient(
					ctx,
					c,
				)
				if err != nil {
					return err
				}

				return contract.AddStorageProviderAction(ctx, client, params)
			},
		},
		{
			Name:    "update-sp",
			Aliases: []string{"us"},
			Usage:   "Update a storage provider in the MarketDealWrapper contract",
			Flags: append(
				commonWriteFlags,
				&cli.Uint64Flag{
					Name:     "actor-id",
					Aliases:  []string{"id"},
					Usage:    "Actor ID of the storage provider",
					Required: true,
				},
				&cli.StringFlag{
					Name:     "eth-addr",
					Aliases:  []string{"e"},
					Usage:    "Ethereum address of the storage provider to receive payments",
					Required: false, // Optional for updates
				},
				&cli.StringFlag{
					Name:     "token",
					Aliases:  []string{"t"},
					Usage:    "ERC20 token address used for payments",
					Required: false, // Optional for updates
				},
				&cli.Float64Flag{
					Name:     "price-per-tb-per-month",
					Aliases:  []string{"p"},
					Usage:    "New price per TB per month",
					Required: false, // Optional for updates
				},
			),
			Action: func(c *cli.Context) error {
				ctx := context.Background()
				params := contract.StorageProviderParams{
					ActorId:              c.Uint64("actor-id"),
					EthAddr:              utils.ParseHexAddress(c.String("eth-addr")),
					Token:                utils.ParseHexAddress(c.String("token")),
					PricePerBytePerEpoch: utils.ConvertPrice(c.Float64("price-per-tb-per-month")),
				}

				client, err := eth.NewETHClient(
					ctx,
					c,
				)
				if err != nil {
					return err
				}

				return contract.UpdateStorageProviderAction(ctx, client, params)
			},
		},
		{
			Name:      "add-to-whitelist",
			Aliases:   []string{"atw"},
			Usage:     "Add an address to the whitelist in the MarketDealWrapper contract. Get ethAddress using fil get-eth-addr command",
			ArgsUsage: "<address>",
			Flags:     commonWriteFlags,

			Action: func(c *cli.Context) error {
				ctx := context.Background()

				// Retrieve the address from arguments
				if c.Args().Len() < 1 {
					return fmt.Errorf("address argument is required")
				}
				address := c.Args().Get(0)

				client, err := eth.NewETHClient(
					ctx,
					c,
				)
				if err != nil {
					return err
				}
				return contract.AddToWhitelistAction(ctx, client, address)
			},
		},
		{
			Name:      "remove-from-whitelist",
			Aliases:   []string{"rfw"},
			Usage:     "Remove an address from the whitelist in the MarketDealWrapper contract. Get ethAddress using fil get-eth-addr command",
			ArgsUsage: "<address>",
			Flags:     commonWriteFlags,
			Action: func(c *cli.Context) error {
				ctx := context.Background()

				// Retrieve the address from arguments
				if c.Args().Len() < 1 {
					return fmt.Errorf("address argument is required")
				}
				address := c.Args().Get(0)

				client, err := eth.NewETHClient(
					ctx,
					c,
				)
				if err != nil {
					return err
				}
				return contract.RemoveFromWhitelistAction(ctx, client, address)
			},
		},
		{
			Name:      "add-funds",
			Aliases:   []string{"af"},
			Usage:     "Add native funds to the MarketDealWrapper contract",
			ArgsUsage: "<amount>",
			Flags:     commonWriteFlags,
			Action: func(c *cli.Context) error {
				ctx := context.Background()
				amount := c.Args().Get(0)
				if amount == "" {
					return fmt.Errorf("missing amount argument")
				}

				client, err := eth.NewETHClient(
					ctx,
					c,
				)
				if err != nil {
					return err
				}

				return contract.AddFundsAction(ctx, client, amount)
			},
		},
		{
			Name:      "withdraw-funds",
			Aliases:   []string{"wf"},
			Usage:     "Withdraw native funds from the MarketDealWrapper contract",
			ArgsUsage: "<amount>",
			Flags:     commonWriteFlags,
			Action: func(c *cli.Context) error {
				ctx := context.Background()
				amount := c.Args().Get(0)
				if amount == "" {
					return fmt.Errorf("missing amount argument")
				}

				client, err := eth.NewETHClient(
					ctx,
					c,
				)
				if err != nil {
					return err
				}

				return contract.WithdrawFundsAction(ctx, client, amount)
			},
		},
		{
			Name:      "approve-erc20",
			Aliases:   []string{"ae"},
			Usage:     "Approve the MarketDealWrapper contract to spend ERC20 tokens",
			ArgsUsage: "<spender> <amount>",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "contract-address",
					Aliases:  []string{"c"},
					Usage:    "Token contract address",
					Required: true,
				},
				&cli.StringFlag{
					Name:    "abi-path",
					Aliases: []string{"b"},
					Usage:   "Path to the contract ABI file",
					Value:   "./contracts/out/MarketDealWrapper.sol/MarketDealWrapper.json",
				},
				&cli.StringFlag{
					Name:    "private-key",
					Aliases: []string{"k"},
					Usage:   "Private key for signing transactions (overrides .env)",
				},
				&cli.StringFlag{
					Name:    "rpc-url",
					Aliases: []string{"r"},
					Usage:   "RPC URL for the Ethereum node (overrides .env)",
				},
			},
			Action: func(c *cli.Context) error {
				ctx := context.Background()

				// Retrieve token address and amount from arguments
				if c.Args().Len() < 2 {
					return fmt.Errorf("spender and amount arguments are required")
				}
				spender := c.Args().Get(0)
				amount := c.Args().Get(1)
				if amount == "" {
					return fmt.Errorf("missing amount argument")
				}

				// Initialize ETH client
				client, err := eth.NewETHClient(
					ctx,
					c,
				)
				if err != nil {
					return err
				}

				// Execute approve action
				return contract.ApproveERC20Action(ctx, client, spender, amount)
			},
		},
		{
			Name:      "add-funds-erc20",
			Aliases:   []string{"afe"},
			Usage:     "Add ERC20 tokens to the MarketDealWrapper contract",
			Flags:     commonWriteFlags,
			ArgsUsage: "<token> <amount>",
			Action: func(c *cli.Context) error {
				ctx := context.Background()
				tokenAddress := c.Args().Get(0)
				amount := c.Args().Get(1)
				if tokenAddress == "" || amount == "" {
					return fmt.Errorf("missing token or amount argument")
				}
				client, err := eth.NewETHClient(ctx, c)
				if err != nil {
					return err
				}
				return contract.AddFundsERC20Action(ctx, client, tokenAddress, amount)
			},
		},
		{
			Name:      "withdraw-funds-erc20",
			Aliases:   []string{"wfe"},
			Usage:     "Withdraw ERC20 tokens from the MarketDealWrapper contract",
			Flags:     commonWriteFlags,
			ArgsUsage: "<token> <amount>",
			Action: func(c *cli.Context) error {
				ctx := context.Background()
				tokenAddress := c.Args().Get(0)
				amount := c.Args().Get(1)
				if tokenAddress == "" || amount == "" {
					return fmt.Errorf("missing token or amount argument")
				}
				client, err := eth.NewETHClient(ctx, c)
				if err != nil {
					return err
				}
				return contract.WithdrawFundsERC20Action(ctx, client, tokenAddress, amount)
			},
		},
		{
			Name:      "withdraw-sp-funds-by-token",
			Aliases:   []string{"wsfbt"},
			Usage:     "Withdraw total SP funds by ERC20 token from the MarketDealWrapper contract",
			Flags:     commonWriteFlags,
			ArgsUsage: "<token>",
			Action: func(c *cli.Context) error {
				ctx := context.Background()
				token := c.Args().Get(0)
				if token == "" {
					return fmt.Errorf("missing token argument")
				}
				client, err := eth.NewETHClient(ctx, c)
				if err != nil {
					return err
				}
				return contract.WithdrawSpFundsByTokenAction(ctx, client, token)
			},
		},
		{
			Name:      "withdraw-sp-funds-for-deal",
			Aliases:   []string{"wsffd"},
			Usage:     "Withdraw SP funds for a specific deal from the MarketDealWrapper contract",
			ArgsUsage: "<deal-id>",
			Flags:     commonWriteFlags,
			Action: func(c *cli.Context) error {
				ctx := context.Background()
				dealIdStr := c.Args().Get(0)
				if dealIdStr == "" {
					return fmt.Errorf("missing deal-id argument")
				}
				dealId, err := strconv.ParseUint(dealIdStr, 10, 64)
				if err != nil {
					return fmt.Errorf("invalid deal-id: %v", err)
				}

				client, err := eth.NewETHClient(
					ctx,
					c,
				)
				if err != nil {
					return err
				}

				return contract.WithdrawSpFundsForDealAction(ctx, client, dealId)
			},
		},
		{
			Name:      "withdraw-sp-funds-for-terminated-deal",
			Aliases:   []string{"wsfftd"},
			Usage:     "Withdraw SP funds for a terminated deal from the MarketDealWrapper contract",
			ArgsUsage: "<deal-id>",
			Flags:     commonWriteFlags,
			Action: func(c *cli.Context) error {
				ctx := context.Background()
				dealIdStr := c.Args().Get(0)
				if dealIdStr == "" {
					return fmt.Errorf("missing deal-id argument")
				}
				dealId, err := strconv.ParseUint(dealIdStr, 10, 64)
				if err != nil {
					return fmt.Errorf("invalid deal-id: %v", err)
				}

				client, err := eth.NewETHClient(
					ctx,
					c,
				)
				if err != nil {
					return err
				}

				return contract.WithdrawSpFundsForTerminatedDealAction(ctx, client, dealId)
			},
		},
	},
}
