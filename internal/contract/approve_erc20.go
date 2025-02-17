package contract

import (
	"context"
	"fmt"
	"math/big"

	"github.com/eastore-project/fil-deal-wrapper/internal/eth"
	"github.com/eastore-project/fil-deal-wrapper/internal/types"
	"github.com/eastore-project/fil-deal-wrapper/internal/utils"

	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
)

// ApproveERC20Action approves the MarketDealWrapper contract to spend a specified amount of ERC20 tokens.
func ApproveERC20Action(ctx context.Context, client *types.ETHClient, spender string, amount string) error {

	// amount is in wei, convert
	weiAmount := new(big.Int)
	_, ok := weiAmount.SetString(amount, 10)
	if !ok {
		return fmt.Errorf("invalid amount: %s", amount)
	}
	fmt.Printf("Adding ERC20 funds: %s Wei\n", weiAmount.String())

	// Define the spender as the MarketDealWrapper contract address
	spenderAddr := common.HexToAddress(spender)

	// ERC20 ABI
	const erc20ABI = `[{"constant":false,"inputs":[{"name":"spender","type":"address"},{"name":"amount","type":"uint256"}],"name":"approve","outputs":[{"name":"","type":"bool"}],"type":"function"}]`

	parsedABI, err := abi.JSON(strings.NewReader(erc20ABI))
	if err != nil {
		return fmt.Errorf("failed to parse ERC20 ABI: %v", err)
	}

	// Prepare transaction input by encoding the approve method and parameters
	input, err := parsedABI.Pack("approve", spenderAddr, weiAmount)
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
		ABI:             parsedABI,
		Method:          "approve",
		Params:          []interface{}{spenderAddr, weiAmount},
		Value:           nil, // No Ether to send
	}

	// Sign and send transaction
	signedTx, err := eth.SignAndSendTransaction(ctx, client.Client, txOpts, input)
	if err != nil {
		return fmt.Errorf("failed to approve ERC20 tokens: %v", err)
	}

	fmt.Printf("ERC20 approval transaction sent. Transaction hash: %s\n", signedTx.Hash().Hex())
	fmt.Println("Waiting for confirmation...")

	// Wait for receipt
	receipt, err := eth.WaitForReceipt(ctx, client.Client, signedTx.Hash())
	if err != nil {
		return fmt.Errorf("failed to get transaction receipt: %v", err)
	}

	if receipt.Status == ethTypes.ReceiptStatusSuccessful {
		fmt.Println("ERC20 approval successful!")
	} else {
		return fmt.Errorf("transaction failed with status: %v", receipt.Status)
	}

	return nil
}
