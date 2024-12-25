package filecoin

import (
	"context"
	"fmt"
	"log"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/filecoin-project/boost/cli/node"
	"github.com/filecoin-project/go-address"
	cborutil "github.com/filecoin-project/go-cbor-util"
	"github.com/filecoin-project/lotus/api"
	"golang.org/x/crypto/blake2b"
)

// GetEthAddr converts a Filecoin address to its corresponding Ethereum address using the Boost node.
func GetEthAddr(ctx context.Context, filecoinAddrStr string, repoPath string) error {
	// Convert string to address.Address
	filecoinAddr, err := address.NewFromString(filecoinAddrStr)
	if err != nil {
		return fmt.Errorf("invalid Filecoin address: %v", err)
	}

	// Set up the Boost node
	n, err := node.Setup(repoPath)
	if err != nil {
		log.Fatalf("Failed to set up Boost node: %v", err)
		return err
	}

	fmt.Println("Using Filecoin Address:", filecoinAddr)

	// Derive the Ethereum address
	ethAddress, err := DeriveEthAddr(ctx, n, filecoinAddr)
	if err != nil {
		return fmt.Errorf("failed to derive Ethereum address: %v", err)
	}

	fmt.Println("Derived Ethereum Address:", ethAddress.Hex())

	return nil
}

func DeriveEthAddr(ctx context.Context, n *node.Node, filecoinAddr address.Address) (common.Address, error) {

	// Create dummy data to sign
	dummyData := "dummy"
	dummyBuf, err := cborutil.Dump(dummyData)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to CBOR dump data: %v", err)
	}

	// Compute Blake2b checksum of the dummy data
	b2sum := blake2b.Sum256(dummyBuf)

	// Sign the dummy data using the Boost wallet
	sig, err := n.Wallet.WalletSign(ctx, filecoinAddr, dummyBuf, api.MsgMeta{Type: api.MTUnknown})
	if err != nil {
		log.Fatalf("Failed to sign data: %v", err)
		return common.Address{}, err
	}

	// Recover the public key from the signature and hash
	pubKey, err := crypto.Ecrecover(b2sum[:], sig.Data)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to recover public key: %v", err)
	}

	// Compute Keccak256 hash of the public key (excluding the first byte 0x04)
	pubKeyHash := crypto.Keccak256(pubKey[1:])

	// Derive the Ethereum address by taking the last 20 bytes of the hash
	ethAddress := common.BytesToAddress(pubKeyHash[12:])

	return ethAddress, nil
}
