package contract

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"wrappedeal/internal/types"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
)

// GetSpFromIdAction performs the CLI action to get storage provider details by Actor ID
func GetSpFromIdAction(ctx context.Context, client *types.ETHClient, actorId uint64) (*StorageProviderParams, error) {
	// Prepare call input
	input, err := client.ContractABI.Pack("getSpFromId", actorId)
	if err != nil {
		return nil, fmt.Errorf("failed to pack parameters: %v", err)
	}

	// Make the call
	callMsg := ethereum.CallMsg{
		To:   &client.ContractAddr,
		Data: input,
	}

	output, err := client.Client.CallContract(ctx, callMsg, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to call contract: %v", err)
	}

	// Unpack the result into StorageProviderParams
	var spParams StorageProviderParams
	// err = client.ContractABI.UnpackIntoInterface(&spParams, "getSpFromId", output)
	result, err := client.ContractABI.Unpack("getSpFromId", output)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack result: %v", err)
	}
	structResult := result[0].(struct {
		ActorId              uint64         `json:"actorId"`
		EthAddr              common.Address `json:"ethAddr"`
		Token                common.Address `json:"token"`
		PricePerBytePerEpoch *big.Int       `json:"pricePerBytePerEpoch"`
	})
	spParams = StorageProviderParams(structResult)
	
    // Marshal the struct into JSON with indentation
    jsonBytes, err := json.MarshalIndent(spParams, "", "  ")
    if err != nil {
        return nil, fmt.Errorf("failed to marshal StorageProviderParams to JSON: %v", err)
    }

    fmt.Println(string(jsonBytes))
	return &spParams, nil
}
