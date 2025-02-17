package contract

import (
	"context"
	"fmt"

	"github.com/eastore-project/fil-deal-wrapper/internal/eth"
	"github.com/eastore-project/fil-deal-wrapper/internal/types"
	"github.com/eastore-project/fil-deal-wrapper/internal/utils"

	ethTypes "github.com/ethereum/go-ethereum/core/types"
)

// WithdrawSpFundsForDealAction withdraws SP funds for a specific deal from the MarketDealWrapper contract
func WithdrawSpFundsForDealAction(ctx context.Context, client *types.ETHClient, dealId uint64) error {
	// Prepare transaction input by encoding the method and parameters
	input, err := client.ContractABI.Pack("withdrawSpFundsForDeal", dealId)
	if err != nil {
		return fmt.Errorf("failed to pack parameters: %v", err)
	}

	// Estimate gas limit
	gasLimit, err := utils.EstimateGas(client.Client, client.FromAddress, client.ContractAddr, input)
	if err != nil {
		return fmt.Errorf("gas estimation failed: %v", err)
	}

	// Create transaction options
	txOpts := types.TransactionOptions{
		FromAddress:     client.FromAddress,
		PrivateKey:      client.PrivateKey,
		GasPrice:        client.GasPrice,
		GasLimit:        gasLimit,
		Nonce:           client.Nonce,
		ChainID:         client.ChainID,
		ContractAddress: client.ContractAddr,
		ABI:             client.ContractABI,
		Method:          "withdrawSpFundsForDeal",
		Params:          []interface{}{dealId},
		Value:           nil, // No Ether to send
	}

	// Sign and send transaction
	signedTx, err := eth.SignAndSendTransaction(ctx, client.Client, txOpts, input)
	if err != nil {
		return fmt.Errorf("failed to withdraw SP funds for deal: %v", err)
	}

	fmt.Printf("Transaction sent: %s\n", signedTx.Hash().Hex())
	fmt.Println("Waiting for confirmation...")

	// Wait for receipt
	receipt, err := eth.WaitForReceipt(ctx, client.Client, signedTx.Hash())
	if err != nil {
		return fmt.Errorf("failed to get transaction receipt: %v", err)
	}

	if receipt.Status == ethTypes.ReceiptStatusSuccessful {
		fmt.Println("SP funds withdrawn for deal successfully!")
	} else {
		fmt.Println("Transaction failed.")
	}

	return nil
}
