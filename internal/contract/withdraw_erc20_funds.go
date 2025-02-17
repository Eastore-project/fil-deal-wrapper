package contract

import (
	"context"
	"fmt"
	"math/big"

	"github.com/eastore-project/fil-deal-wrapper/internal/eth"
	"github.com/eastore-project/fil-deal-wrapper/internal/types"
	"github.com/eastore-project/fil-deal-wrapper/internal/utils"

	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
)

// WithdrawFundsERC20Action withdraws ERC20 tokens from the MarketDealWrapper contract
func WithdrawFundsERC20Action(ctx context.Context, client *types.ETHClient, tokenAddress string, amount string) error {
	// amount is in wei, convert
	weiAmount := new(big.Int)
	_, ok := weiAmount.SetString(amount, 10)
	if !ok {
		return fmt.Errorf("invalid amount: %s", amount)
	}
	fmt.Printf("Withdrawing ERC20 funds: %s Wei\n", weiAmount.String())

	// Get the ERC20 token contract address
	token := common.HexToAddress(tokenAddress)

	// Prepare transaction input by encoding the method and parameters
	input, err := client.ContractABI.Pack("withdrawFundsERC20", token, weiAmount)
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
		Method:          "withdrawFundsERC20",
		Params:          []interface{}{token, weiAmount},
		Value:           nil, // No Ether to send
	}

	// Sign and send transaction
	signedTx, err := eth.SignAndSendTransaction(ctx, client.Client, txOpts, input)
	if err != nil {
		return fmt.Errorf("failed to withdraw ERC20 funds: %v", err)
	}

	fmt.Printf("ERC20 funds withdrawn. Transaction hash: %s\n", signedTx.Hash().Hex())
	fmt.Println("Waiting for confirmation...")

	// Wait for receipt
	receipt, err := eth.WaitForReceipt(ctx, client.Client, signedTx.Hash())
	if err != nil {
		return fmt.Errorf("failed to get transaction receipt: %v", err)
	}

	if receipt.Status == ethTypes.ReceiptStatusSuccessful {
		fmt.Println("ERC20 funds withdrawn successfully!")
	} else {
		fmt.Println("Transaction failed.")
	}

	return nil
}
