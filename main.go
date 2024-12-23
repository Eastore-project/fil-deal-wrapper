package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"github.com/filecoin-project/boost/api"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/urfave/cli/v2"
)

// NewWalletClient initializes a new WalletAPI client.
func NewWalletClient(ctx context.Context, rpcURL string) (api.Wallet, jsonrpc.ClientCloser, error) {
    var res api.WalletStruct
    closer, err := jsonrpc.NewMergeClient(ctx, rpcURL, "Wallet", api.GetInternalStructs(&res), nil)
    if err != nil {
        return nil, nil, err 
    }
    return &res, closer, nil
}

func main() {
    app := &cli.App{
        Name:  "boost-cli",
        Usage: "CLI tool to interact with Boost for wallet management and message signing",
        Commands: []*cli.Command{
            {
                Name:  "list-wallets",
                Usage: "List all configured wallet addresses",
                Flags: []cli.Flag{
                    &cli.StringFlag{
                        Name:    "rpc-url",
                        Usage:   "Boost RPC endpoint URL",
                        Value:   "https://api.calibration.node.glif.io", // Default RPC URL; adjust if necessary
                        EnvVars: []string{"BOOST_RPC_URL"},
                    },
                    &cli.BoolFlag{
                        Name:    "json",
                        Usage:   "Output results in JSON format",
                        Aliases: []string{"j"},
                    },
                },
                Action: func(cctx *cli.Context) error {
                    rpcURL := cctx.String("rpc-url")
                    jsonOutput := cctx.Bool("json")

                    // Initialize Wallet API client
                    ctx := context.Background()
                    walletAPI, closer, err := NewWalletClient(ctx, rpcURL)
                    if err != nil {
                        return fmt.Errorf("failed to create Wallet API client: %v", err)
                    }
                    defer closer()

                    // Retrieve list of wallets
                    wallets, err := walletAPI.WalletList(ctx)
                    if err != nil {
                        return fmt.Errorf("failed to list wallets: %v", err)
                    }

                    // Output the list
                    if jsonOutput {
                        output, err := json.MarshalIndent(wallets, "", "  ")
                        if err != nil {
                            return fmt.Errorf("failed to marshal JSON: %v", err)
                        }
                        fmt.Println(string(output))
                    } else {
                        fmt.Println("Configured Wallet Addresses:")
                        for _, addr := range wallets {
                            fmt.Println(addr.String())
                        }
                    }

                    return nil
                },
            },
            {
                Name:      "sign",
                Usage:     "Sign a message using a specified wallet",
                ArgsUsage: "<wallet-address> <hex-message>",
                Flags: []cli.Flag{
                    &cli.StringFlag{
                        Name:    "rpc-url",
                        Usage:   "Boost RPC endpoint URL",
                        Value:   "http://127.0.0.1:1234/rpc/v0", // Default RPC URL; adjust if necessary
                        EnvVars: []string{"BOOST_RPC_URL"},
                    },
                    &cli.BoolFlag{
                        Name:    "json",
                        Usage:   "Output signature in JSON format",
                        Aliases: []string{"j"},
                    },
                },
                Action: func(cctx *cli.Context) error {
                    if cctx.NArg() != 2 {
                        return fmt.Errorf("usage: boost-cli sign <wallet-address> <hex-message>")
                    }

                    walletAddrStr := cctx.Args().Get(0)
                    hexMsg := cctx.Args().Get(1)
                    rpcURL := cctx.String("rpc-url")
                    jsonOutput := cctx.Bool("json")

                    // Parse the wallet address
                    walletAddr, err := address.NewFromString(walletAddrStr)
                    if err != nil {
                        return fmt.Errorf("invalid wallet address: %v", err)
                    }

                    // Decode the hex-encoded message
                    msg, err := hex.DecodeString(hexMsg)
                    if err != nil {
                        return fmt.Errorf("invalid hex message: %v", err)
                    }

                    // Initialize Wallet API client
                    ctx := context.Background()
                    walletAPI, closer, err := NewWalletClient(ctx, rpcURL)
                    if err != nil {
                        return fmt.Errorf("failed to create Wallet API client: %v", err)
                    }
                    defer closer()

                    // Sign the message
                    sig, err := walletAPI.WalletSign(ctx, walletAddr, msg)
                    if err != nil {
                        return fmt.Errorf("failed to sign message: %v", err)
                    }

                    // Prepare signature bytes
                    sigBytes := append([]byte{byte(sig.Type)}, sig.Data...)
                    sigHex := hex.EncodeToString(sigBytes)

                    // Output the signature
                    if jsonOutput {
                        output := map[string]string{
                            "signature": sigHex,
                        }
                        jsonOutputStr, err := json.MarshalIndent(output, "", "  ")
                        if err != nil {
                            return fmt.Errorf("failed to marshal JSON: %v", err)
                        }
                        fmt.Println(string(jsonOutputStr))
                    } else {
                        fmt.Printf("Signature: %s\n", sigHex)
                    }

                    return nil
                },
            },
        },
    }

    // Run the CLI application
    err := app.Run(os.Args)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}