package wordlist

import (
	"context"
	"fmt"
)

// Get will retrieve all the names from remote or local sources
func Get(ctx context.Context, names ...string) (ret []WordlistMetadata, err error) {
	if err := CreateLocalCache(); err != nil {
		return nil, fmt.Errorf("failed to create local cache dir: %w", err)
	}

	// don't download existing things in the cache
	local, err := GetLocalDirListing()
	if err != nil {
		return nil, fmt.Errorf("failed to get local dir listing: %w", err)
	}

	got := make(map[string]WordlistMetadata)
	for _, v := range local {
		got[v.Filename] = v
		got[v.Shortname] = v
	}

	missing := make([]string, 0)
	for _, v := range names {
		if w, ok := got[v]; ok {
			ret = append(ret, w)
			continue
		}
		missing = append(missing, v)
	}

	if len(missing) == 0 {
		return ret, nil
	}

	// Check against the remote list of what we can fetch
	remote, err := GetRemoteWordlists()
	if err != nil {
		return ret, fmt.Errorf("failed to get remote wordlists")
	}

	remotegot := make(map[string]WordlistMetadata)
	for _, v := range remote {
		remotegot[v.Filename] = v
		remotegot[v.Shortname] = v
	}

	nonExist := make([]string, 0)
	for _, v := range missing {
		if vv, ok := remotegot[v]; ok {
			ret = append(ret, vv)
			continue
		}
		nonExist = append(nonExist, v)
	}
	if len(nonExist) != 0 {
		return ret, fmt.Errorf("invalid names: %v", nonExist)
	}

	return ret, err
}
