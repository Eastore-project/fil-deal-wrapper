package filecoin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"wrappedeal/internal/utils"

	lcli "github.com/filecoin-project/lotus/cli"
	"github.com/urfave/cli/v2"
)


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
	result, err := carParams.GenerateCarUtil()
	if err != nil {
		return fmt.Errorf("failed to generate CAR: %v", err)
	}

	commp := result.PieceCid
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
	} else {
		httpURL = cctx.String("http-url")
	}
	// delete the local car file
	err = os.Remove(carFilePath)
	if err != nil {
		return fmt.Errorf("failed to delete local car file: %v", err)
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
		result.PieceSize,
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
	return nil
}
