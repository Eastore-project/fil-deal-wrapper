package filecoin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/eastore-project/fil-deal-wrapper/internal/utils"

	"github.com/ethereum/go-ethereum/common"
	clinode "github.com/filecoin-project/boost/cli/node"
	"github.com/filecoin-project/boost/cmd"
	"github.com/filecoin-project/boost/storagemarket/types"
	types2 "github.com/filecoin-project/boost/transport/types"
	"github.com/filecoin-project/go-address"
	cborutil "github.com/filecoin-project/go-cbor-util"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/builtin"
	"github.com/filecoin-project/go-state-types/builtin/v9/market"
	"github.com/filecoin-project/lotus/api"
	chain_types "github.com/filecoin-project/lotus/chain/types"
	lcli "github.com/filecoin-project/lotus/cli"
	"github.com/google/uuid"
	"github.com/ipfs/go-cid"
	inet "github.com/libp2p/go-libp2p/core/network"
	"github.com/urfave/cli/v2"
)

const DealProtocolv120 = "/fil/storage/mk/1.2.0"

func DealCmdAction(cctx *cli.Context, isOnline bool) error {
	ctx := context.Background()
	api, closer, err := lcli.GetGatewayAPI(cctx)
	if err != nil {
		return fmt.Errorf("cant setup gateway connection: %w", err)
	}
	defer closer()

	err = MakeDeal(
		ctx,
		api,
		cctx.String("repo"),
		cctx.String("wallet"),
		cctx.String("commp"),
		cctx.String("payload-cid"),
		cctx.String("http-url"),
		cctx.String("provider"),
		cctx.Uint64("piece-size"),
		cctx.Uint64("car-size"),
		cctx.Int("start-epoch-head-offset"),
		cctx.Int("start-epoch"),
		cctx.Int("duration"),
		cctx.Int64("provider-collateral"),
		cctx.Int64("storage-price"),
		cctx.Bool("verified"),
		cctx.Bool("remove-unsealed-copy"),
		cctx.Bool("skip-ipni-announce"),
		isOnline,
		cctx.StringSlice("http-headers"),
		cctx.String("contract"),
	)
	if err != nil {
		return fmt.Errorf("deal failed: %w", err)
	}

	return nil
}

func MakeDeal(
	ctx context.Context,
	api api.Gateway,
	repo string,
	wallet string,
	commp string,
	payloadCidStr string,
	url string,
	provider string,
	pieceSize uint64,
	carSize uint64,
	startEpochHeadOffset int,
	startEpoch int,
	duration int,
	providerCollateral int64,
	storagePrice int64,
	verified bool,
	removeUnsealedCopy bool,
	skipIPNIAnnounce bool,
	isOnline bool,
	httpHeaders []string,
	contract string,
) error {
	n, err := clinode.Setup(repo)
	if err != nil {
		return err
	}

	walletAddr, err := n.GetProvidedOrDefaultWallet(ctx, wallet)
	if err != nil {
		return err
	}

	fmt.Println("selected ", "wallet", walletAddr)

	maddr, err := address.NewFromString(provider)
	if err != nil {
		return err
	}

	addrInfo, err := cmd.GetAddrInfo(ctx, api, maddr)
	if err != nil {
		return err
	}

	fmt.Println("found storage provider", "id", addrInfo.ID, "multiaddrs", addrInfo.Addrs, "addr", maddr)

	if err := n.Host.Connect(ctx, *addrInfo); err != nil {
		return fmt.Errorf("failed to connect to peer %s: %w", addrInfo.ID, err)
	}

	x, err := n.Host.Peerstore().FirstSupportedProtocol(addrInfo.ID, DealProtocolv120)
	if err != nil {
		return fmt.Errorf("getting protocols for peer %s: %w", addrInfo.ID, err)
	}

	if len(x) == 0 {
		return fmt.Errorf("boost client cannot make a deal with storage provider %s because it does not support protocol version 1.2.0", maddr)
	}

	dealUuid := uuid.New()

	pieceCid, err := cid.Parse(commp)
	if err != nil {
		return fmt.Errorf("parsing commp '%s': %w", commp, err)
	}

	rootCid, err := cid.Parse(payloadCidStr)
	if err != nil {
		return fmt.Errorf("parsing payload cid %s: %w", payloadCidStr, err)
	}
	transfer := types.Transfer{}
	if isOnline {
		if carSize == 0 {
			return fmt.Errorf("size of car file cannot be 0")
		}

		transfer.Size = carSize
		// Store the path to the CAR file as a transfer parameter
		transferParams := &types2.HttpRequest{URL: url}

		if url != "" {
			transferParams.Headers = make(map[string]string)

			for _, header := range httpHeaders {
				sp := strings.Split(header, "=")
				if len(sp) != 2 {
					return fmt.Errorf("malformed http header: %s", header)
				}

				transferParams.Headers[sp[0]] = sp[1]
			}
		}

		paramsBytes, err := json.Marshal(transferParams)
		if err != nil {
			return fmt.Errorf("marshalling request parameters: %w", err)
		}
		transfer.Type = "http"
		transfer.Params = paramsBytes
	}

	var providerCollateralAmount abi.TokenAmount
	if providerCollateral == 0 {
		bounds, err := api.StateDealProviderCollateralBounds(ctx, abi.PaddedPieceSize(pieceSize), verified, chain_types.EmptyTSK)
		if err != nil {
			return fmt.Errorf("node error getting collateral bounds: %w", err)
		}

		providerCollateralAmount = big.Div(big.Mul(bounds.Min, big.NewInt(6)), big.NewInt(5)) // add 20%
	} else {
		providerCollateralAmount = abi.NewTokenAmount(providerCollateral)
	}

	tipset, err := api.ChainHead(ctx)
	if err != nil {
		return fmt.Errorf("cannot get chain head: %w", err)
	}

	head := tipset.Height()

	if startEpochHeadOffset > 0 && startEpoch > 0 {
		return errors.New("only one flag from `start-epoch-head-offset' or `start-epoch` can be specified")
	}

	var effectiveStartEpoch abi.ChainEpoch
	if startEpochHeadOffset > 0 {
		effectiveStartEpoch = head + abi.ChainEpoch(startEpochHeadOffset)
	} else if startEpoch > 0 {
		effectiveStartEpoch = abi.ChainEpoch(startEpoch)
	} else {
		effectiveStartEpoch = head + abi.ChainEpoch(5760) // head + 2 days
	}

	ethAddr := common.HexToAddress(contract)
	filClient, err := address.NewDelegatedAddress(builtin.EthereumAddressManagerActorID, ethAddr[:])
	if err != nil {
		return fmt.Errorf("failed to translate onramp address (%s) into a "+
			"Filecoin f4 address: %w", ethAddr, err)
	}

	// put the signer's actorId in label for signer verification in contract
	signerActorId, err := api.StateLookupID(ctx, walletAddr, chain_types.EmptyTSK)
	if err != nil {
		return fmt.Errorf("failed to lookup actorId for signer address: %w", err)
	}
	stringActorId, err := utils.GetStringActorId(signerActorId)
	if err != nil {
		return fmt.Errorf("failed to convert actorId to string: %w", err)
	}

	label, err := market.NewLabelFromString(stringActorId)
	if err != nil {
		return fmt.Errorf("failed to create label: %w", err)
	}

	// Create a deal proposal to storage provider using deal protocol v1.2.0 format
	dealProposal, err := DealProposal(
		ctx, n, filClient, walletAddr, abi.PaddedPieceSize(pieceSize), pieceCid, maddr, label, effectiveStartEpoch, duration, verified, providerCollateralAmount, abi.NewTokenAmount(storagePrice),
	)
	if err != nil {
		return fmt.Errorf("failed to create a deal proposal: %w", err)
	}

	dealParams := types.DealParams{
		DealUUID:           dealUuid,
		ClientDealProposal: *dealProposal,
		DealDataRoot:       rootCid,
		IsOffline:          !isOnline,
		Transfer:           transfer,
		RemoveUnsealedCopy: removeUnsealedCopy,
		SkipIPNIAnnounce:   skipIPNIAnnounce,
	}

	fmt.Println("about to submit deal proposal", "uuid", dealUuid.String())

	s, err := n.Host.NewStream(ctx, addrInfo.ID, DealProtocolv120)
	if err != nil {
		return fmt.Errorf("failed to open stream to peer %s: %w", addrInfo.ID, err)
	}
	defer s.Close()

	var resp types.DealResponse
	if err := doRpc(ctx, s, &dealParams, &resp); err != nil {
		return fmt.Errorf("send proposal rpc: %w", err)
	}

	if !resp.Accepted {
		return fmt.Errorf("deal proposal rejected: %s", resp.Message)
	}

	msg := "sent deal proposal"
	if !isOnline {
		msg += " for offline deal"
	}
	msg += "\n"
	msg += fmt.Sprintf("  deal uuid: %s\n", dealUuid)
	msg += fmt.Sprintf("  storage provider: %s\n", maddr)
	msg += fmt.Sprintf("  client contract address: %s\n", filClient)
	msg += fmt.Sprintf("  signer wallet: %s\n", walletAddr)
	msg += fmt.Sprintf("  payload cid: %s\n", rootCid)
	if isOnline {
		msg += fmt.Sprintf("  url: %s\n", url)
	}
	msg += fmt.Sprintf("  commp: %s\n", dealProposal.Proposal.PieceCID)
	msg += fmt.Sprintf("  start epoch: %d\n", dealProposal.Proposal.StartEpoch)
	msg += fmt.Sprintf("  end epoch: %d\n", dealProposal.Proposal.EndEpoch)
	msg += fmt.Sprintf("  provider collateral: %s\n", chain_types.FIL(dealProposal.Proposal.ProviderCollateral).Short())
	fmt.Println(msg)

	return nil
}

func DealProposal(ctx context.Context, n *clinode.Node, clientAddr address.Address, signerAddr address.Address, pieceSize abi.PaddedPieceSize, pieceCid cid.Cid, minerAddr address.Address, label market.DealLabel, startEpoch abi.ChainEpoch, duration int, verified bool, providerCollateral abi.TokenAmount, storagePrice abi.TokenAmount) (*market.ClientDealProposal, error) {
	endEpoch := startEpoch + abi.ChainEpoch(duration)
	// deal proposal expects total storage price for deal per epoch, therefore we
	// multiply pieceSize * storagePrice (which is set per epoch per GiB) and divide by 2^30
	storagePricePerEpochForDeal := big.Div(big.Mul(big.NewInt(int64(pieceSize)), storagePrice), big.NewInt(int64(1<<30)))

	proposal := market.DealProposal{
		PieceCID:             pieceCid,
		PieceSize:            pieceSize,
		VerifiedDeal:         verified,
		Client:               clientAddr,
		Provider:             minerAddr,
		Label:                label,
		StartEpoch:           startEpoch,
		EndEpoch:             endEpoch,
		StoragePricePerEpoch: storagePricePerEpochForDeal,
		ProviderCollateral:   providerCollateral,
	}

	buf, err := cborutil.Dump(&proposal)
	if err != nil {
		return nil, err
	}

	sig, err := n.Wallet.WalletSign(ctx, signerAddr, buf, api.MsgMeta{Type: api.MTDealProposal})
	if err != nil {
		return nil, fmt.Errorf("wallet sign failed: %w", err)
	}

	return &market.ClientDealProposal{
		Proposal:        proposal,
		ClientSignature: *sig,
	}, nil
}

func doRpc(ctx context.Context, s inet.Stream, req interface{}, resp interface{}) error {
	errc := make(chan error)
	go func() {
		if err := cborutil.WriteCborRPC(s, req); err != nil {
			errc <- fmt.Errorf("failed to send request: %w", err)
			return
		}

		if err := cborutil.ReadCborRPC(s, resp); err != nil {
			errc <- fmt.Errorf("failed to read response: %w", err)
			return
		}

		errc <- nil
	}()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
