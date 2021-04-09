package wordlist

import (
	"context"
	"fmt"

	"github.com/assetnote/kiterunner/pkg/log"
	"github.com/hashicorp/go-multierror"
)

// Save will save the specified names to the local cache.
// TODO: implement special case for "all"
func Save(ctx context.Context, nocache bool, names ...string) (ret []WordlistMetadata, err error) {
	if len(names) == 0 {
		return nil, fmt.Errorf("no names provided")
	}

	wms, err := Get(ctx, names...)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve wordlist metadata: %w", err)
	}

	canDownload := make([]WordlistMetadata, 0)
	for _, v := range wms {
		if v.Cached && !nocache {
			continue
		}
		canDownload = append(canDownload, v)
	}

	// download wordlists in parallel
	errChan := make(chan error, len(canDownload))
	for _, v := range canDownload {
		go func(w WordlistMetadata) {
			err := w.Cache()
			if err == nil {
				log.Info().Str("path", w.LocalFilenamePanic()).Str("name", w.Shortname).Msg("file saved")
			}
			errChan <- err
		}(v)
	}

	var merr *multierror.Error
	for i := 0; i < len(canDownload); i++ {
		select {
		case err := <-errChan:
			if err != nil {
				merr = multierror.Append(merr, err)
			}
		case <-ctx.Done():
			break
		}
	}
	log.Info().Int("files", len(canDownload)).Msg("completed caching")

	return wms, merr.ErrorOrNil()
}

func CheckAllCached(in []WordlistMetadata) ([]WordlistMetadata, error) {
	local, err := GetLocalDirListing()
	if err != nil {
		return in, fmt.Errorf("failed to get local listing: %w", err)
	}
	ret := make([]WordlistMetadata, 0)

	exist := make(map[string]interface{})
	for _, v := range local {
		exist[v.Filename] = struct{}{}
	}

	for _, v := range in {
		if _, ok := exist[v.Filename]; ok {
			v.Cached = true
		}
		ret = append(ret, v)
	}
	return ret, nil
}
