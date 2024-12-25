package filecoin

import (
	"context"
	"fmt"
	"math/bits"
	"os"
	"path/filepath"

	"wrappedeal/internal/utils"

	"github.com/filecoin-project/go-state-types/abi"
	lcli "github.com/filecoin-project/lotus/cli"
	"github.com/urfave/cli/v2"
)

var localDealFlags = []cli.Flag{
	&cli.StringFlag{
		Name:     "path",
		Usage:    "Path to the file or folder to make the deal",
		Required: true,
	},
	&cli.BoolFlag{
		Name:  "lighthouse",
		Usage: "Use Lighthouse as a buffer",
		Value: false,
	},
	&cli.StringFlag{
		Name:  "apikey",
		Usage: "API key for Lighthouse (overrides .env)",
	},
	&cli.StringFlag{
		Name:     "provider",
		Usage:    "storage provider on-chain address",
		Required: true,
	},
	&cli.StringFlag{
		Name:  "payload-cid",
		Usage: "root CID of the CAR file",
	},
	&cli.IntFlag{
		Name:  "start-epoch-head-offset",
		Usage: "start epoch by when the deal should be proved by provider on-chain after current chain head",
	},
	&cli.IntFlag{
		Name:  "start-epoch",
		Usage: "start epoch by when the deal should be proved by provider on-chain",
	},
	&cli.IntFlag{
		Name:  "duration",
		Usage: "duration of the deal in epochs",
		Value: 518400, // default is 2880 * 180 == 180 days
	},
	&cli.IntFlag{
		Name:  "provider-collateral",
		Usage: "deal collateral that storage miner must put in escrow; if empty, the min collateral for the given piece size will be used",
	},
	&cli.Int64Flag{
		Name:  "storage-price",
		Usage: "storage price in attoFIL per epoch per GiB",
		Value: 1,
	},
	&cli.BoolFlag{
		Name:  "verified",
		Usage: "whether the deal funds should come from verified client data-cap",
		Value: true,
	},
	&cli.BoolFlag{
		Name:  "remove-unsealed-copy",
		Usage: "indicates that an unsealed copy of the sector in not required for fast retrieval",
		Value: false,
	},
	&cli.StringFlag{
		Name:  "wallet",
		Usage: "wallet address to be used to initiate the deal",
	},
	&cli.BoolFlag{
		Name:  "skip-ipni-announce",
		Usage: "indicates that deal index should not be announced to the IPNI(Network Indexer)",
		Value: false,
	},
	&cli.StringFlag{
		Name:  "http-url",
		Usage: "http url to CAR file",
	},
	&cli.StringSliceFlag{
		Name:  "http-headers",
		Usage: "http headers to be passed with the request (e.g key=value)",
	},
	&cli.Uint64Flag{
		Name:  "piece-size",
		Usage: "size of the CAR file as a padded piece",
	},
	&cli.StringFlag{
		Name:  "out-dir",
		Usage: "output directory for CAR files",
		Value: "/tmp",
	},
	&cli.StringFlag{
		Name:     "repo",
		Aliases:  []string{"R"},
		Usage:    "Boost client repository directory path (default: ~/.boost-client)",
		Value:    "~/.boost-client",
		Required: false,
	},
	&cli.StringFlag{
		Name:     "contract",
		Aliases:  []string{"c"},
		Usage:    "contract address to make deal with",
		Required: true,
	},
}

var LocalDealCmd = &cli.Command{
	Name:  "local-deal",
	Usage: "Make a local deal with wrappedeal",
	Flags: localDealFlags,
	Action: func(cctx *cli.Context) error {
		return LocalDealCmdAction(cctx, true)

	},
}

func LocalDealCmdAction(cctx *cli.Context, isOnline bool) error {
	ctx := context.Background()

	api, closer, err := lcli.GetGatewayAPI(cctx)
	if err != nil {
		return fmt.Errorf("cant setup gateway connection: %w", err)
	}
	defer closer()

	// Retrieve flags
	path := cctx.String("path")
	useLighthouse := cctx.Bool("lighthouse")
	apiKey := cctx.String("apikey")
	pieceSize := cctx.Uint64("piece-size")
	// return error if pieceSize is not a power of 2
	if pieceSize != 0 && (pieceSize&(pieceSize-1)) != 0 {
		return fmt.Errorf("piece-size must be a power of 2")
	}

	// Validate the provided path
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("the provided path does not exist: %s", path)
	}

	if pieceSize == 0 {
		// get the file/folder size at the path location
		size, err := utils.GetSize(path)
		if err != nil {
			return fmt.Errorf("failed to get size of file/folder: %v", err)
		}
		next := 1 << (64 - bits.LeadingZeros64(size+256))
		pieceSize = uint64(abi.PaddedPieceSize(next))

	}

	outDir := cctx.String("out-dir")

	// Generate CAR file
	carParams := utils.CarParams{
		Input:     path,
		OutDir:    outDir,
		Single:    true,
		PieceSize: pieceSize, // Will be set based on generated CAR
		Parent:    path,
		TmpDir:    "",
	}
	result, err := carParams.GenerateCar()
	if err != nil {
		return fmt.Errorf("failed to generate CAR: %v", err)
	}

	//		Ipld:      ipld,
	// DataCid:   cid,
	// PieceCid:  commCid.String(),
	// PieceSize: pieceSize,
	// CidMap:    cidMap,

	commp := result.PieceCid
	fmt.Println("Piece CID:", commp)
	// Assuming carSize is the byte size of the generated CAR file
	carFilePath := filepath.Join(outDir, fmt.Sprintf("%s.car", commp))
	carFileInfo, err := os.Stat(carFilePath)
	if err != nil {
		return fmt.Errorf("failed to retrieve CAR file info: %v", err)
	}
	carSize := uint64(carFileInfo.Size())

	var httpURL string
	if useLighthouse {
		if apiKey == "" {
			apiKey = os.Getenv("LIGHTHOUSE_API_KEY")
			if apiKey == "" {
				return fmt.Errorf("API key for Lighthouse is required when using --lighthouse flag")
			}
		}
		uploadResp, err := utils.UploadToLighthouse(carFilePath, apiKey)
		if err != nil {
			return fmt.Errorf("failed to upload to Lighthouse: %v", err)
		}
		httpURL = fmt.Sprintf("https://gateway.lighthouse.storage/ipfs/%s", uploadResp.Hash)
		fmt.Printf("Car Uploaded to Lighthouse: %s\n", httpURL)
		// delete the local car file
		err = os.Remove(carFilePath)
		if err != nil {
			return fmt.Errorf("failed to delete local car file: %v", err)
		}
	} else {
		httpURL = cctx.String("http-url")
	}

	err = MakeDeal(
		ctx,
		api,
		cctx.String("repo"),
		cctx.String("wallet"),
		commp,
		result.DataCid,
		httpURL,
		cctx.String("provider"),
		pieceSize,
		carSize,
		cctx.Int("start-epoch-head-offset"),
		cctx.Int("start-epoch"),
		cctx.Int("duration"),
		cctx.Int64("provider-collateral"),
		cctx.Int64("storage-price"),
		cctx.Bool("verified"),
		cctx.Bool("remove-unsealed-copy"),
		cctx.Bool("skip-ipni-announce"),
		true,
		cctx.StringSlice("http-headers"),
		cctx.String("contract"),
	)
	if err != nil {
		return fmt.Errorf("deal failed: %w", err)
	}
	// Call DealCmdAction with the enriched context
	return nil
}
