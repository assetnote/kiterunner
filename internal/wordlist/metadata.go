package wordlist

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/assetnote/kiterunner/pkg/log"
	"github.com/assetnote/kiterunner/pkg/proute"
)

type WordlistMetadata struct {
	Shortname string  `json:"Shortname,omitempty"`
	Filename  string  `json:"Filename,omitempty"`
	LineCount int     `json:"Line Count,omitempty"`
	FileSize  string  `json:"File Size,omitempty"`
	Date      float64 `json:"Date,omitempty"`
	Download  string  `json:"Download,omitempty"`
	Source    string  `json:"Source,omitempty"`
	Cached    bool    `json:"Cached"`
}

func GetRemoteWordlists() ([]WordlistMetadata, error) {
	ret := make([]WordlistMetadata, 0)

	for _, source := range wordlists {
		resp, err := http.Get(RemoteWordlistBase + source)
		if err != nil {
			return nil, fmt.Errorf("failed to get url: %w", err)
		}
		defer resp.Body.Close()

		type data struct {
			Data []WordlistMetadata `json:"data"`
		}

		var jsond data
		if err := json.NewDecoder(resp.Body).Decode(&jsond); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}

		for _, v := range jsond.Data {
			v.Download = getDownloadLink(v.Download)
			v.Source = source
			v.Shortname = nameConvert(v.Filename)
			ret = append(ret, v)
		}
	}

	return ret, nil
}

func (w WordlistMetadata) LocalFilenamePanic() string {
	dir, err := GetLocalDir()
	if err != nil {
		panic(fmt.Errorf("failed to get local dir: %w", err))
	}

	outfile := path.Join(dir, w.Filename)
	return outfile
}

func (w WordlistMetadata) LocalKiteFilename() (string, error) {
	dir, err := GetLocalDir()
	if err != nil {
		return "", fmt.Errorf("failed to get local dir: %w", err)
	}

	fn := fmt.Sprintf("%s.kite", strings.Replace(w.Filename, path.Ext(w.Filename), "", -1))
	outfile := path.Join(dir, fn)
	return outfile, nil
}

func (w WordlistMetadata) LocalFilename() (string, error) {
	dir, err := GetLocalDir()
	if err != nil {
		return "", fmt.Errorf("failed to get local dir: %w", err)
	}

	outfile := path.Join(dir, w.Filename)
	return outfile, nil
}

// Cache will downlad the file and save it to disk
func (w *WordlistMetadata) Cache() error {
	// if we're already cached do nothing
	if w.Cached {
		log.Info().Str("wordlist", w.Filename).Str("local", w.LocalFilenamePanic()).Msg("already cached")
		return nil
	}
	if w.Download == "" {
		return fmt.Errorf("invalid wordlist download url")
	}
	log.Info().Str("wordlist", w.Filename).Str("Url", w.Download).Str("local", w.LocalFilenamePanic()).Msg("caching locally")
	resp, err := http.Get(w.Download)
	if err != nil {
		return fmt.Errorf("failed to get wordlist: %w", err)
	}
	defer resp.Body.Close()

	outfile, err := w.LocalFilename()
	if err != nil {
		return fmt.Errorf("failed to get local filename: %w", err)
	}

	output, err := os.OpenFile(outfile, os.O_CREATE|os.O_RDWR, os.FileMode(0644))
	if err != nil {
		return fmt.Errorf("failed to get local file: %w", err)
	}
	defer output.Close()

	_, err = io.Copy(output, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to copy response into cache: %w", err)
	}

	w.Cached = true
	return nil
}

// API will atttempt to load a compiled kite API from disk.
// If this doesnt exist, then we attempt to compile a local file to disk.
// If this local file doesnt exist, then we download it first and cache it
func (w WordlistMetadata) APIS() (ret proute.APIS, err error) {
	// get the localfile first
	localkite, err := w.LocalKiteFilename()
	if err != nil {
		return ret, fmt.Errorf("failed to get local filename: %w", err)
	}

	ret, err = proute.DecodeAPIProtoFile(localkite)
	if err == nil {
		log.Debug().Str("wordlist", w.Filename).Str("local", localkite).Msg("loaded compiled kite from cache")
		return ret, nil
	}
	// if there's any error, we'll just try and re-compile the wordlist

	// file does not exist so we should try and compile the existing wordlist
	localfile, err := w.LocalFilename()
	if err != nil {
		return ret, fmt.Errorf("failed to get local filename: %w", err)
	}

	// Cache will not perform any operation if its already cached
	if err := w.Cache(); err != nil {
		return ret, fmt.Errorf("failed to cache file: %w", err)
	}

	words, err := w.Words()
	if err != nil {
		return ret, fmt.Errorf("failed to read words from file: %w", err)
	}

	log.Debug().Int("lines", len(words)).Str("name", w.Filename).Msg("loaded file from disk")
	api, err := proute.FromStringSlice(words, localfile)
	if err != nil {
		return ret, fmt.Errorf("failed to convert to kite: %w", err)
	}
	ret = append(ret, api)

	// cache the compiled file for later
	if err := ret.EncodeProtoFile(localkite); err != nil {
		return ret, fmt.Errorf("failed to cache localkite: %w", err)
	}

	return ret, nil
}

// words will attempt to retrieve the words from the cache. If it fails, it will
func (w WordlistMetadata) Words() ([]string, error) {
	localfile, err := w.LocalFilename()
	if err != nil {
		return nil, fmt.Errorf("failed to get local filename: %w", err)
	}

	f, err := os.Open(localfile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// cache the file if it doesnt exist yet
			if err := w.Cache(); err != nil {
				return nil, fmt.Errorf("failed to load words when caching file: %w", err)
			}
			f, err = os.Open(localfile)
			if err != nil {
				return nil, fmt.Errorf("failed to load words after caching file: %w", err)
			}
		} else {
			return nil, fmt.Errorf("unexpected err when opening file: %w", err)
		}
	}
	defer f.Close()

	ret := make([]string, 0)
	r := bufio.NewScanner(f)
	for r.Scan() {
		ret = append(ret, strings.TrimSpace(r.Text()))
	}
	return ret, nil
}

// Cached performs a single stat operation against the local directory to see if the file exists
// on disk. This is not efficient, but a few dozen stat calls never hurt anyone.
// If you want to go fast, use CheckAllCached
func (w *WordlistMetadata) IsCached() bool {
	ret, err := GetLocalDir()
	if err != nil {
		return w.Cached
	}

	fullfile := filepath.Join(ret, w.Filename)
	if _, err := os.Stat(fullfile); err == nil {
		w.Cached = true
	} else if os.IsNotExist(err) {
		w.Cached = false
	} else {
		log.Error().Err(err).Msg("failed to check cache for file")
	}
	return w.Cached
}
