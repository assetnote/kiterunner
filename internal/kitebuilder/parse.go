package kitebuilder

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	errors2 "github.com/assetnote/kiterunner/pkg/errors"
	"github.com/assetnote/kiterunner/pkg/log"
	"github.com/assetnote/kiterunner/pkg/kitebuilder"
	"github.com/assetnote/kiterunner/pkg/proute"
	"github.com/hashicorp/go-multierror"
)

type ScanOptions struct {
	Debug bool
}

func NewDefaultScanOptions() *ScanOptions {
	return &ScanOptions{}
}

func Debug(enabled bool) ScanOption {
	return func(o *ScanOptions) {
		o.Debug = enabled
	}
}

type ScanOption func(o *ScanOptions)

func ScanStdin(ctx context.Context, opts ...ScanOption) error {
	return DebugPrintReader(ctx, os.Stdin, opts...)
}

func ScanFile(ctx context.Context, filename string, opts ...ScanOption) error {
	jsonFile, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer jsonFile.Close()
	return DebugPrintReader(ctx, jsonFile, opts...)
}

func fixOutputFilename(filename string) string {
	if !strings.HasSuffix(filename, ".kite") {
		log.Info().Str("filename", filename).Msg(".kite extension added to filename")
		return fmt.Sprintf("%s.kite", filename)
	}
	return filename
}

func CompileFile(ctx context.Context, input string, outputFile string, opts ...ScanOption) error {
	jsonFile, err := os.Open(input)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer jsonFile.Close()
	return Compile(ctx, jsonFile, outputFile, opts...)
}

func Compile(ctx context.Context, r io.Reader, outputFile string, opts ...ScanOption) error {
	// add the .kite extension if it doesnt exist
	outputFile = fixOutputFilename(outputFile)

	data, err := ioutil.ReadAll(r)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var merr *multierror.Error
	api, err := kitebuilder.SlowLoadJSONBytes(data)
	if errors.As(err, &merr) {
		for _, v := range merr.Errors {
			errors2.PrintError(v, 0)
		}
	} else if err != nil {
		return fmt.Errorf("failed to parse json: %w", err)
	}

	apis, err := proute.FromKitebuilderAPIs(api)
	if errors.As(err, &merr) {
		log.Error().Msg("errors while parsing apis")
		for _, v := range merr.Errors {
			errors2.PrintError(v, 0)
		}
	} else if err != nil {
		return fmt.Errorf("failed to parse api: %w", err)
	}

	if err := proute.APIS(apis).EncodeProtoFile(outputFile); err != nil {
		return fmt.Errorf("failed to encode apis: %w", err)
	}

	return nil
}

func DebugPrintReader(ctx context.Context, r io.Reader, opts ...ScanOption) error {
	options := NewDefaultScanOptions()
	for _, o := range opts {
		o(options)
	}

	data, err := ioutil.ReadAll(r)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	if options.Debug {
		return DebugPrintBytes(data)
	} else {
		return PrintBytes(data)
	}
}

func DebugPrintBytes(data []byte) error {
	log.Debug().Msg("debug printing")

	var merr *multierror.Error
	api, err := kitebuilder.SlowLoadJSONBytes(data)
	if errors.As(err, &merr) {
		for _, v := range merr.Errors {
			errors2.PrintError(v, 0)
		}
	} else if err != nil {
		return fmt.Errorf("failed to parse json: %w", err)
	}

	// kitebuilder.PrintAPIs(api)
	// for _, v := range api {
	//	pr := proute.FromKitebuilderAPI(v)
	//	pr.DebugPrint()
	// }
	for _, v := range api {
		tmp, err := proute.FromKitebuilderAPI(v)
		if errors.As(err, &merr) {
			log.Error().Str("id", v.ID).Msg("errors while parsing api")
			for _, v := range merr.Errors {
				errors2.PrintError(v, 1)
			}
		} else if err != nil {
			return fmt.Errorf("failed to parse api: %w", err)
		}

		wcr, err := proute.ToKiterunnerRoutes(tmp)
		if errors.As(err, &merr) {
			log.Error().Str("id", v.ID).Msg("errors while building routes")
			for _, v := range merr.Errors {
				errors2.PrintError(v, 1)
			}
		} else if err != nil {
			return fmt.Errorf("failed to parse api: %w", err)
		}
		for _, v := range wcr {
			_ = v
			// log.Log().Object("route", v)
		}
	}
	return nil
}

func PrintBytes(data []byte) error {
	api, err := kitebuilder.LoadJSONBytes(data)
	if err != nil {
		return fmt.Errorf("failed to parse json: %w", err)
	}

	kitebuilder.PrintAPIs(api)
	return nil
}
