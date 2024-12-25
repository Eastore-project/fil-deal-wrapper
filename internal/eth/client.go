package eth

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"wrappedeal/internal/types"
	"wrappedeal/internal/utils"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
	"github.com/urfave/cli/v2"
)

// NewETHClient initializes and returns a new ETHClient
func NewETHClient(ctx context.Context, c *cli.Context) (*types.ETHClient, error) {

	// Parse flags
	rpcURL := c.String("rpc-url")
	privateKeyHex := c.String("private-key")
	contractAddress := c.String("contract-address")
	abiPath := c.String("abi-path")

	// Load environment variables from .env
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found or error reading .env")
	}

	// Use rpcURL from flag or environment
	if rpcURL == "" {
		rpcURL = os.Getenv("RPC_URL")
		if rpcURL == "" {
			return nil, fmt.Errorf("RPC URL must be provided via flag or .env")
		}
	}

	// Use private key from flag or environment
	if privateKeyHex == "" {
		privateKeyHex = os.Getenv("ETH_PRIVATE_KEY")
		if privateKeyHex == "" {
			return nil, fmt.Errorf("private key must be provided via flag or .env")
		}
	}

	// Connect to Ethereum node
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum node: %v", err)
	}

	// Load and parse ABI
	contractABI, err := loadABI(abiPath)
	if err != nil {
		return nil, err
	}

	// Load private key
	privateKey, fromAddress, err := loadPrivateKey(privateKeyHex)
	if err != nil {
		return nil, err
	}

	// Parse contract address
	contractAddr := common.HexToAddress(contractAddress)

	// Get nonce
	nonce := utils.GetNonce(client, fromAddress)

	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to suggest gas price: %v", err)
	}

	// Get chain ID
	chainID, err := client.NetworkID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get network ID: %v", err)
	}

	return &types.ETHClient{
		Client:       client,
		PrivateKey:   privateKey,
		FromAddress:  fromAddress,
		ContractABI:  contractABI,
		ContractAddr: contractAddr,
		ChainID:      chainID,
		Nonce:        nonce,
		GasPrice:     gasPrice,
	}, nil
}

// loadABI loads and parses the ABI from the given path
func loadABI(abiPath string) (abi.ABI, error) {
	absoluteABIPath, err := filepath.Abs(abiPath)
	if err != nil {
		return abi.ABI{}, fmt.Errorf("failed to get absolute path of ABI: %v", err)
	}
	abiFile, err := os.ReadFile(absoluteABIPath)
	if err != nil {
		return abi.ABI{}, fmt.Errorf("failed to read ABI file: %v", err)
	}

	// Unmarshal ABIWrapper to extract the "abi" field
	var abiWrapper types.ABIWrapper
	if err := json.Unmarshal(abiFile, &abiWrapper); err != nil {
		return abi.ABI{}, fmt.Errorf("failed to unmarshal ABI JSON: %v", err)
	}

	// Parse ABI
	parsedABI, err := abi.JSON(bytes.NewReader(abiWrapper.ABI))
	if err != nil {
		return abi.ABI{}, fmt.Errorf("failed to parse ABI: %v", err)
	}

	return parsedABI, nil
}

// loadPrivateKey loads the private key and returns the key and the corresponding address
func loadPrivateKey(privateKeyHex string) (*ecdsa.PrivateKey, common.Address, error) {
	// Remove "0x" prefix if present
	privateKeyHex = removeHexPrefix(privateKeyHex)

	// Load private key
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, common.Address{}, fmt.Errorf("invalid private key: %v", err)
	}

	// Derive public address
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, common.Address{}, fmt.Errorf("error casting public key to ECDSA")
	}
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	return privateKey, fromAddress, nil
}

// removeHexPrefix removes the "0x" prefix if present
func removeHexPrefix(s string) string {
	if len(s) >= 2 && s[:2] == "0x" {
		return s[2:]
	}
	return s
}

// ConvertEtherToWei converts ether to wei
func ConvertEtherToWei(ether float64) *big.Int {
	wei := new(big.Float).Mul(big.NewFloat(ether), big.NewFloat(math.Pow10(18)))
	weiInt, _ := wei.Int(nil)
	return weiInt
}
