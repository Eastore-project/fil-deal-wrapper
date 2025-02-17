package contract

import (
	"context"
	"fmt"

	"github.com/eastore-project/fil-deal-wrapper/internal/types"

	"math/big"

	"github.com/ethereum/go-ethereum"
)

// GetSpFundsForDealAction retrieves the currently claimable SP funds for a specific deal
func GetSpFundsForDealAction(ctx context.Context, client *types.ETHClient, dealId uint64) error {
	// Prepare call input
	input, err := client.ContractABI.Pack("getSpFundsForDeal", dealId)
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

	// Unpack the result into a uint256
	var funds *big.Int
	err = client.ContractABI.UnpackIntoInterface(&funds, "getSpFundsForDeal", output)
	if err != nil {
		return fmt.Errorf("failed to unpack result: %v", err)
	}

	fmt.Printf("Currently claimable SP funds for Deal ID %d: %s\n", dealId, funds.String())
	return nil
}
