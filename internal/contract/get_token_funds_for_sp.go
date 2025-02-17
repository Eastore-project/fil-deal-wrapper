package contract

import (
	"context"
	"fmt"

	"math/big"

	"github.com/eastore-project/fil-deal-wrapper/internal/types"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
)

// GetTokenFundsForSPAction retrieves the currently claimable SP funds for a specific ERC20 token and actor ID
func GetTokenFundsForSPAction(ctx context.Context, client *types.ETHClient, tokenAddress string, actorId uint64) error {
	// Convert string token address to common.Address
	token := common.HexToAddress(tokenAddress)

	// Prepare call input
	input, err := client.ContractABI.Pack("getTokenFundsForSp", token, actorId)
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
	err = client.ContractABI.UnpackIntoInterface(&funds, "getTokenFundsForSp", output)
	if err != nil {
		return fmt.Errorf("failed to unpack result: %v", err)
	}

	fmt.Printf("Currently claimable SP funds for Token %s and Actor ID %d: %s\n", token.Hex(), actorId, funds.String())
	return nil
}
