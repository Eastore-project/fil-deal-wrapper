package contract

import (
	"context"
	"fmt"

	"github.com/eastore-project/fil-deal-wrapper/internal/eth"
	"github.com/eastore-project/fil-deal-wrapper/internal/types"
	"github.com/eastore-project/fil-deal-wrapper/internal/utils"

	"math/big"

	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
)

// UpdateStorageProviderAction performs the CLI action to update a storage provider
func UpdateStorageProviderAction(ctx context.Context, client *types.ETHClient, params StorageProviderParams) error {
	// If EthAddr or Token is not provided, fetch existing storage provider details
	if params.EthAddr == (common.Address{}) || params.Token == (common.Address{}) {
		spDetails, err := GetSpFromIdAction(ctx, client,  params.ActorId)
		if err != nil {
			return fmt.Errorf("failed to fetch existing storage provider details: %v", err)
		}
		if params.EthAddr == (common.Address{}) {
			params.EthAddr = spDetails.EthAddr
		}
		if params.Token == (common.Address{}) {
			params.Token = spDetails.Token
		}
		if params.PricePerBytePerEpoch == nil {
			params.PricePerBytePerEpoch = spDetails.PricePerBytePerEpoch
		}
	}

	// Prepare transaction input
	input, err := client.ContractABI.Pack("updateStorageProvider",
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
		Method:          "updateStorageProvider",
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
		fmt.Println("Storage provider updated successfully!")
	} else {
		fmt.Println("Transaction failed.")
	}

	return nil
}
