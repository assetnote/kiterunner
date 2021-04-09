package kitebuilder

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	errors2 "github.com/assetnote/kiterunner/pkg/errors"
	"github.com/assetnote/kiterunner/pkg/kitebuilder"
	"github.com/assetnote/kiterunner/pkg/log"
	"github.com/assetnote/kiterunner/pkg/proute"
	"github.com/hashicorp/go-multierror"
)

func ConvertFiles(ctx context.Context, input string, output string) error {
	inType, err := FileTypeFromExtension(input)
	if err != nil {
		return fmt.Errorf("input error: %w", err)
	}

	infile, err := os.Open(input)
	if err != nil {
		return fmt.Errorf("input file error: %w", err)
	}
	defer infile.Close()

	outType, err := FileTypeFromExtension(output)
	if err != nil {
		return fmt.Errorf("output error: %w", err)
	}

	outfile, err := os.OpenFile(output, os.O_CREATE|os.O_RDWR, os.FileMode(0666))
	if err != nil {
		return fmt.Errorf("output file error: %w", err)
	}
	defer outfile.Close()

	log.Info().Str("input", input).
		Str("input-type", inType.String()).
		Str("output", output).
		Str("output-type", outType.String()).
		Msg("converting")

	if err := convertFile(infile, inType, input, outfile, outType); err != nil {
		return fmt.Errorf("conversion error: %w", err)
	}

	return nil
}

//go:generate stringer -type=FileType
type FileType int

const (
	UNKNOWN FileType = iota
	TXT
	JSON
	KITE
)

var (
	ErrUnsupportedFileType  = errors.New("Unsupported filetype. Only supported extensions: txt, json, kite")
	ErrSameTypeNoConversion = errors.New("input and output filetype the same. no conversion performed")
)

func FileTypeFromExtension(filename string) (FileType, error) {
	ext := strings.Split(filename, ".")
	switch strings.ToLower(ext[len(ext)-1]) {
	case "txt":
		return TXT, nil
	case "json":
		return JSON, nil
	case "kite":
		return KITE, nil
	}

	return UNKNOWN, ErrUnsupportedFileType
}

// convertFile will convert everything into our intermediate proute format, then output the format in the desired output
// the inputType and outputType must be different. If they're the same this will error (since no conversion should occur)
func convertFile(input io.Reader, inputType FileType, inputFilename string, output io.Writer, outputType FileType) error {
	if inputType == outputType {
		return ErrSameTypeNoConversion
	}

	var inAPI proute.APIS

	switch inputType {
	case TXT:
		ret, err := proute.FromStringSliceReader(input, inputFilename)
		if err != nil {
			return fmt.Errorf("parsing txt input error: %w", err)
		}
		inAPI = append(inAPI, ret)
	case KITE:
		ret, err := proute.DecodeProtoAPI(input)
		if err != nil {
			return fmt.Errorf("parsing kite input error: %w", err)
		}
		inAPI = ret
	case JSON:
		gotSchema, err := kitebuilder.LoadJSONReader(input)
		if err != nil {
			return fmt.Errorf("parsing kitebuilder json input error: %w", err)
		}
		var merr *multierror.Error
		inAPI, err = proute.FromKitebuilderAPIs(gotSchema)
		if err != nil && !errors.As(err, &merr) {
			return fmt.Errorf("parsing kitebuilder json input error: %w", err)
		}
	default:
		return fmt.Errorf("unexpected input filetype: %s %w", inputType, ErrUnsupportedFileType)
	}

	// we should now have our format encoded in the inAPI

	switch outputType {
	case TXT:
		if err := inAPI.EncodeStringSlice(output); err != nil {
			var merr *multierror.Error
			if errors.As(err, &merr) {
				for _, v := range merr.Errors {
					errors2.PrintError(v, 0)
				}
			} else {
				return fmt.Errorf("converting to txt output error: %w", err)
			}
		}
	case KITE:
		if err := inAPI.EncodeProto(output); err != nil {
			return fmt.Errorf("converting to kite output error: %w", err)
		}
	case JSON:
		kb, err := inAPI.ToKiteBuilderAPIS()
		if err != nil {
			return fmt.Errorf("converting to json output error: %w", err)
		}

		if err := json.NewEncoder(output).Encode(kb); err != nil {
			return fmt.Errorf("converting to json output encoding error: %w", err)
		}
	}

	return nil
}
