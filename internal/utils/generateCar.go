package utils

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path"
	"path/filepath"
	"wrappedeal/internal/types"

	commcid "github.com/filecoin-project/go-fil-commcid"
	commp "github.com/filecoin-project/go-fil-commp-hashhash"
	"github.com/google/uuid"
)
type Result struct {
	Ipld      *FsNode
	DataCid   string
	PieceCid  string
	PieceSize uint64
	CidMap    map[string]CidMapValue
	CarSize   uint64
}
type CarParams types.CarParams

const BufSize = (4 << 20) / 128 * 127

func (c *CarParams) GenerateCarUtil() (Result, error) {
	ctx := context.Background()
	var input []Finfo
	if c.Single {
		stat, err := os.Stat(c.Input)
		if err != nil {
			return Result{}, err
		}
		if stat.IsDir() {
			err := filepath.Walk(c.Input, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info.IsDir() {
					return nil
				}
				input = append(input, Finfo{
					Path:  path,
					Size:  info.Size(),
					Start: 0,
					End:   info.Size(),
				})
				return nil
			})
			if err != nil {
				return Result{}, err
			}
		} else {
			input = append(input, Finfo{
				Path:  c.Input,
				Size:  stat.Size(),
				Start: 0,
				End:   stat.Size(),
			})
		}
	} else {
		var inputBytes []byte
		if c.Input == "-" {
			reader := bufio.NewReader(os.Stdin)
			buf := new(bytes.Buffer)
			_, err := buf.ReadFrom(reader)
			if err != nil {
				return Result{}, err
			}
			inputBytes = buf.Bytes()
		} else {
			bytes, err := os.ReadFile(c.Input)
			if err != nil {
				return Result{}, err
			}
			inputBytes = bytes
		}
		err := json.Unmarshal(inputBytes, &input)
		if err != nil {
			return Result{}, err
		}
	}

	outFilename := uuid.New().String() + ".car"
	outPath := path.Join(c.OutDir, outFilename)
	carF, err := os.Create(outPath)
	if err != nil {
		return Result{}, err
	}
	cp := new(commp.Calc)
	writer := bufio.NewWriterSize(io.MultiWriter(carF, cp), BufSize)
	ipld, cid, cidMap, err := GenerateCar(ctx, input, c.Parent, c.TmpDir, writer)
	if err != nil {
		return Result{}, err
	}
	err = writer.Flush()
	if err != nil {
		return Result{}, err
	}
	err = carF.Close()
	if err != nil {
		return Result{}, err
	}
	rawCommP, pieceSize, err := cp.Digest()
	if err != nil {
		return Result{}, err
	}
	if c.PieceSize > 0 {
		rawCommP, err = commp.PadCommP(
			rawCommP,
			pieceSize,
			c.PieceSize,
		)
		if err != nil {
			return Result{}, err
		}
		pieceSize = c.PieceSize
	}
	commCid, err := commcid.DataCommitmentV1ToCID(rawCommP)
	if err != nil {
		return Result{}, err
	}
	err = os.Rename(outPath, path.Join(c.OutDir, commCid.String()+".car"))
	if err != nil {
		return Result{}, err
	}
	result := Result{
		Ipld:      ipld,
		DataCid:   cid,
		PieceCid:  commCid.String(),
		PieceSize: pieceSize,
		CidMap:    cidMap,
	}
	return result, nil
}
