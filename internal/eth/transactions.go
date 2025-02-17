package eth

import (
	"context"
	"fmt"
	"time"

	"github.com/eastore-project/fil-deal-wrapper/internal/types"

	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// SignAndSendTransaction signs and sends the transaction
func SignAndSendTransaction(ctx context.Context, client *ethclient.Client, opts types.TransactionOptions, input []byte) (*ethTypes.Transaction, error) {
	tx := ethTypes.NewTransaction(
		opts.Nonce,
		opts.ContractAddress,
		opts.Value,
		opts.GasLimit,
		opts.GasPrice,
		input,
	)

	signedTx, err := ethTypes.SignTx(tx, ethTypes.NewEIP155Signer(opts.ChainID), opts.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %v", err)
	}

	err = client.SendTransaction(ctx, signedTx)
	if err != nil {
		return nil, fmt.Errorf("failed to send transaction: %v", err)
	}

	return signedTx, nil
}

// WaitForReceipt waits until the transaction receipt is available
func WaitForReceipt(ctx context.Context, client *ethclient.Client, txHash common.Hash) (*ethTypes.Receipt, error) {
	for {
		receipt, err := client.TransactionReceipt(ctx, txHash)
		if err == nil {
			return receipt, nil
		}
		time.Sleep(1 * time.Second)
	}
}
