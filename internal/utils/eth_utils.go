package utils

import (
	"context"
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Utility function to estimate gas
func EstimateGas(client *ethclient.Client, from common.Address, to common.Address, data []byte) (uint64, error) {
	msg := ethereum.CallMsg{
		From: from,
		To:   &to,
		Data: data,
	}
	gasLimit, err := client.EstimateGas(context.Background(), msg)
	if err != nil {
		return 0, fmt.Errorf("failed to estimate gas: %v", err)
	}
	return gasLimit, nil
}

// Utility function to get nonce
func GetNonce(client *ethclient.Client, from common.Address) uint64 {
	nonce, err := client.PendingNonceAt(context.Background(), from)
	if err != nil {
		log.Fatalf("Failed to get nonce: %v", err)
	}
	return nonce
}

// convertPrice converts price per TB per month to price per byte per epoch (30 seconds)
func ConvertPrice(pricePerTbMonth float64) *big.Int {
	// 1 TB = 1e12 bytes
	// 1 month ≈ 30 days
	// 1 epoch = 30 seconds
	bytesInTb := 1024 * 1024 * 1024 * 1024
	epochsPerMonth := 30 * 24 * 60 * 2 // 30 days, 24h, 60m, every 30 seconds

	pricePerBytePerEpoch := (pricePerTbMonth * 1e18) / (float64(bytesInTb) * float64(epochsPerMonth))
	return big.NewInt(int64(pricePerBytePerEpoch))
}

func EncodeAddress(ethAddress common.Address) ([]byte, error) {
	addressType, err := abi.NewType("address", "", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create address type: %w", err)
	}

	// Define the ABI encoding
	arguments := abi.Arguments{
		{Type: addressType},
	}

	// Encode the Ethereum address
	encodedBytes, err := arguments.Pack(ethAddress)
	if err != nil {
		return nil, fmt.Errorf("error encoding address: %w", err)
	}

	return encodedBytes, nil
}

// Helper function to parse Ethereum addresses safely
func ParseHexAddress(addr string) common.Address {
	if addr == "" {
		return common.Address{}
	}
	return common.HexToAddress(addr)
}
