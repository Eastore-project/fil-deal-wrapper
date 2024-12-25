package filecoin

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/filecoin-project/boost/cli/node"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/api"
)

func SignDataAction(ctx context.Context, walletInput string, repoPath string) error {

	// Set up the Boost node
	n, err := node.Setup(repoPath)
	if err != nil {
		log.Fatalf("Failed to set up Boost node: %v", err)
		return err
	}

	// Fetch wallet
	var walletAddr address.Address
	if walletInput == "" {
		walletAddr, err = n.Wallet.GetDefault()
		if err != nil {
			return err
		}
	} else {
		walletAddr, err = address.NewFromString(walletInput)
		if err != nil {
			log.Fatalf("Failed to parse wallet address: %v", err)
			return err
		}
	}
	fmt.Println("Using wallet:", walletAddr)

	// Dummy data
	data := []byte("Hello, Filecoin!")

	// Sign data
	sig, err := n.Wallet.WalletSign(ctx, walletAddr, data, api.MsgMeta{Type: api.MTUnknown})
	if err != nil {
		log.Fatalf("Failed to sign data: %v", err)
		return err
	}
	sigBytes := append([]byte{byte(sig.Type)}, sig.Data...)

	fmt.Println("Signature:", hex.EncodeToString(sigBytes))
	return nil
}
