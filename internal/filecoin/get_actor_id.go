package filecoin

import (
	"context"
	"fmt"
	"log"

	"github.com/eastore-project/fil-deal-wrapper/internal/utils"

	"github.com/filecoin-project/boost/cli/node"
	chain_types "github.com/filecoin-project/lotus/chain/types"
	lcli "github.com/filecoin-project/lotus/cli"
	"github.com/urfave/cli/v2"
)

// GetEthAddr converts a Filecoin address to its corresponding Ethereum address using the Boost node.
func GetActorIdAction(cctx *cli.Context) error {

	ctx := context.Background()
	filecoinAddrStr := cctx.String("filecoin-addr")
	repoFlag := cctx.String("repo")

	// Expand the repo path
	repoPath, err := utils.ExpandPath(repoFlag)
	if err != nil {
		return fmt.Errorf("failed to expand repo path: %v", err)
	}

	api, closer, err := lcli.GetGatewayAPI(cctx)
	if err != nil {
		return fmt.Errorf("cant setup gateway connection: %w", err)
	}
	defer closer()
	
	// Set up the Boost node
	n, err := node.Setup(repoPath)
	if err != nil {
		log.Fatalf("Failed to set up Boost node: %v", err)
		return err
	}

	walletAddr, err := n.GetProvidedOrDefaultWallet(ctx, filecoinAddrStr)
	if err != nil {
		return err
	}

	fmt.Println("Using Filecoin Address:", walletAddr.String())

	signerActorId, err := api.StateLookupID(ctx, walletAddr, chain_types.EmptyTSK)
	if err != nil {
		return fmt.Errorf("failed to lookup actorId for signer address: %w", err)
	}
	stringActorId, err := utils.GetStringActorId(signerActorId)
	if err != nil {
		return fmt.Errorf("failed to convert actorId to string: %w", err)
	}

	fmt.Println("Actor ID:", stringActorId)

	return nil
}
