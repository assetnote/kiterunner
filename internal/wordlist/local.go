package wordlist

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"

	"github.com/assetnote/kiterunner/pkg/log"
	humanize "github.com/dustin/go-humanize"
)


func GetLocalDirPanic() (string) {
	ret, err := GetLocalDir()
	if err != nil {
		panic(err)
	}
	return ret
}

func GetLocalDir() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("failed to get user: %w", err)
	}
	dir := usr.HomeDir
	newpath := filepath.Join(dir, ".cache", "kiterunner", "wordlists")
	return newpath, err
}

func CreateLocalCache() error {
	newpath, err := GetLocalDir()
	if err != nil {
		return fmt.Errorf("failed to get local dir: %w", err)
	}
	log.Debug().Str("path", newpath).Msg("creating directory")
	if err := os.MkdirAll(newpath, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create dir: %w", err)
	}

	return nil
}

func GetLocalDirListing() ([]WordlistMetadata, error) {
	dir, err := GetLocalDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get local dir: %w", err)
	}

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read dir: %w", err)
	}

	ret := make([]WordlistMetadata, 0)
	for _, v := range files {
		r := WordlistMetadata{
			Shortname: nameConvert(v.Name()),
			Filename: v.Name(),
			Cached: true,
			FileSize: humanize.Bytes(uint64(v.Size())),
			Source: "local",
		}
		ret = append(ret, r)
	}

	return ret, nil
}