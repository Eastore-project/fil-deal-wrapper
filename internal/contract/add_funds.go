package contract

import (
	"context"
	"fmt"
	"math/big"

	"wrappedeal/internal/eth"
	"wrappedeal/internal/types"
	"wrappedeal/internal/utils"

	ethTypes "github.com/ethereum/go-ethereum/core/types"
)

// AddFundsAction adds Ether funds to the MarketDealWrapper contract
func AddFundsAction(ctx context.Context, client *types.ETHClient, amount string) error {

	// amount is in wei, convert
	weiAmount := new(big.Int)
	_, ok := weiAmount.SetString(amount, 10)
	if !ok {
		return fmt.Errorf("invalid amount: %s", amount)
	}
	fmt.Printf("Adding funds: %s Wei\n", weiAmount.String())

	// Prepare transaction input (no parameters for addFunds)
	input, err := client.ContractABI.Pack("addFunds")
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
		Method:          "addFunds",
		Params:          []interface{}{},
		Value:           weiAmount,
	}

	// Sign and send transaction
	signedTx, err := eth.SignAndSendTransaction(ctx, client.Client, txOpts, input)
	if err != nil {
		return fmt.Errorf("failed to add funds: %v", err)
	}

	fmt.Printf("Funds added. Transaction hash: %s\n", signedTx.Hash().Hex())
	fmt.Println("Waiting for confirmation...")

	// Wait for receipt
	receipt, err := eth.WaitForReceipt(ctx, client.Client, signedTx.Hash())
	if err != nil {
		return fmt.Errorf("failed to get transaction receipt: %v", err)
	}

	fmt.Println(receipt)

	if receipt.Status == ethTypes.ReceiptStatusSuccessful {
		fmt.Println("Funds added successfully!")
	} else {
		fmt.Println("Transaction failed.")
	}

	return nil
}
