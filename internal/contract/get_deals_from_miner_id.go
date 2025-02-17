package contract

import (
	"context"
	"fmt"

	"github.com/eastore-project/fil-deal-wrapper/internal/types"

	"github.com/ethereum/go-ethereum"
)

// GetDealsFromMinerIdAction retrieves deal IDs associated with a given miner ID
func GetDealsFromMinerIdAction(ctx context.Context, client *types.ETHClient, minerId uint64) error {
	// Prepare call input
	input, err := client.ContractABI.Pack("getDealsFromMinerId", minerId)
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

	// Unpack the result into a slice of uint64
	var dealIds []uint64
	err = client.ContractABI.UnpackIntoInterface(&dealIds, "getDealsFromMinerId", output)
	if err != nil {
		return fmt.Errorf("failed to unpack result: %v", err)
	}

	fmt.Printf("Deal IDs for Miner ID %d: %v\n", minerId, dealIds)
	return nil
}
