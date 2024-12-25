package types

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// StorageProviderParams holds parameters for a storage provider
type StorageProviderParams struct {
	ActorId              *big.Int       `json:"actorId"`
	EthAddr              common.Address `json:"ethAddr"`
	Token                common.Address `json:"token"`
	PricePerBytePerEpoch *big.Int       `json:"pricePerBytePerEpoch"`
}

// GetSpFromIdParams holds parameters for retrieving a storage provider by Actor ID
type GetSpFromIdParams struct {
	ActorId *big.Int
}
