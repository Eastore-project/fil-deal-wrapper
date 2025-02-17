package contract

import (
	"context"
	"fmt"

	"github.com/eastore-project/fil-deal-wrapper/internal/eth"
	"github.com/eastore-project/fil-deal-wrapper/internal/types"
	"github.com/eastore-project/fil-deal-wrapper/internal/utils"

	ethTypes "github.com/ethereum/go-ethereum/core/types"
)

// AddToWhitelistAction adds an address to the whitelist in the MarketDealWrapper contract
func AddToWhitelistAction(ctx context.Context, client *types.ETHClient, actorId uint64) error {

	// Prepare transaction input by encoding the method and parameters
	input, err := client.ContractABI.Pack("addToWhitelist", actorId)
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
		Method:          "addToWhitelist",
		Params:          []interface{}{actorId},
		Value:           nil, // No Ether to send
	}

	// Sign and send transaction
	signedTx, err := eth.SignAndSendTransaction(ctx, client.Client, txOpts, input)
	if err != nil {
		return fmt.Errorf("failed to add address to whitelist: %v", err)
	}

	fmt.Printf("Transaction sent: %s\n", signedTx.Hash().Hex())
	fmt.Println("Waiting for confirmation...")

	// Wait for receipt
	receipt, err := eth.WaitForReceipt(ctx, client.Client, signedTx.Hash())
	if err != nil {
		return fmt.Errorf("failed to get transaction receipt: %v", err)
	}

	if receipt.Status == ethTypes.ReceiptStatusSuccessful {
		fmt.Println("Address added to whitelist successfully!")
	} else {
		fmt.Println("Transaction failed.")
	}

	return nil
}
