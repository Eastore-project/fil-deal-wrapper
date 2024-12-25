package contract

import (
	"context"
	"fmt"

	"wrappedeal/internal/eth"
	"wrappedeal/internal/types"
	"wrappedeal/internal/utils"

	"math/big"

	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
)

// StorageProviderParams holds parameters for updating a storage provider
type StorageProviderParams struct {
	ActorId              uint64         `json:"actorId"`
	EthAddr              common.Address `json:"ethAddr"`
	Token                common.Address `json:"token"`
	PricePerBytePerEpoch *big.Int       `json:"pricePerBytePerEpoch"`
}

// AddStorageProviderAction performs the CLI action to add a storage provider
func AddStorageProviderAction(ctx context.Context, client *types.ETHClient, params StorageProviderParams) error {

	// Prepare transaction input
	input, err := client.ContractABI.Pack("addStorageProvider",
		params.ActorId,
		params.EthAddr,
		params.Token,
		params.PricePerBytePerEpoch,
	)
	if err != nil {
		return fmt.Errorf("failed to pack parameters: %v", err)
	}

	// Estimate gas limit
	gasLimit, err := utils.EstimateGas(client.Client, client.FromAddress, client.ContractAddr, input)
	if err != nil {
		return err
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
		Method:          "addStorageProvider",
		Params:          []interface{}{params.ActorId, params.EthAddr, params.Token, params.PricePerBytePerEpoch},
		Value:           big.NewInt(0),
	}

	// Sign and send transaction
	signedTx, err := eth.SignAndSendTransaction(ctx, client.Client, txOpts, input)
	if err != nil {
		return err
	}

	fmt.Printf("Transaction sent: %s\n", signedTx.Hash().Hex())
	fmt.Println("Waiting for confirmation...")

	// Wait for receipt
	receipt, err := eth.WaitForReceipt(ctx, client.Client, signedTx.Hash())
	if err != nil {
		return fmt.Errorf("failed to get transaction receipt: %v", err)
	}

	if receipt.Status == ethTypes.ReceiptStatusSuccessful {
		fmt.Println("Storage provider added successfully!")
	} else {
		fmt.Println("Transaction failed.")
	}

	return nil
}
