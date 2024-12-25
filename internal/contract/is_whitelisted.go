package contract

import (
	"context"
	"fmt"
	"wrappedeal/internal/types"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
)

// IsWhitelistedAction checks if a given address is whitelisted
func IsWhitelistedAction(ctx context.Context, client *types.ETHClient, address string) error {
	// Convert string address to common.Address
	addr := common.HexToAddress(address)

	// Prepare call input
	input, err := client.ContractABI.Pack("isWhitelisted", addr)
	if err != nil {
		return fmt.Errorf("failed to pack parameters: %v", err)
	}

	// Make the call
	callMsg := ethereum.CallMsg{
		To:   &client.ContractAddr,
		Data: input,
	}

	output, err := client.Client.CallContract(ctx, callMsg, nil)
	if err != nil {
		return fmt.Errorf("failed to call contract: %v", err)
	}

	// Unpack the result into a boolean
	var isWhitelisted bool
	err = client.ContractABI.UnpackIntoInterface(&isWhitelisted, "isWhitelisted", output)
	if err != nil {
		return fmt.Errorf("failed to unpack result: %v", err)
	}

	fmt.Printf("Is address %s whitelisted? %t\n", address, isWhitelisted)
	return nil
}
