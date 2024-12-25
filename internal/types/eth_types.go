package types

import (
	"crypto/ecdsa"
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// TransactionOptions holds necessary fields for a transaction
type TransactionOptions struct {
	FromAddress     common.Address
	PrivateKey      *ecdsa.PrivateKey
	GasPrice        *big.Int
	GasLimit        uint64
	Nonce           uint64
	ChainID         *big.Int
	ContractAddress common.Address
	ABI             abi.ABI
	Method          string
	Params          []interface{}
	Value           *big.Int
}

// ETHClient wraps the Ethereum client instance and holds the private key and address
type ETHClient struct {
	Client       *ethclient.Client
	PrivateKey   *ecdsa.PrivateKey
	FromAddress  common.Address
	ContractABI  abi.ABI
	ContractAddr common.Address
	ChainID      *big.Int
	Nonce        uint64
	GasPrice     *big.Int
}

// ABIWrapper represents the structure of the ABI JSON file
type ABIWrapper struct {
	ABI json.RawMessage `json:"abi"`
}
