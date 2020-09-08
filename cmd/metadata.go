package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/montag451/go-pypi-mirror/metadata"
	"github.com/montag451/go-pypi-mirror/pkg"
)

func createMetadataFiles(dir string, overwrite bool) error {
	if overwrite {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(path, metadata.FileExt) {
				err := os.Remove(path)
				if err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	pkgs, err := pkg.List(dir, true)
	if err != nil {
		return err
	}
	for _, pkg := range pkgs {
		metadataFile := pkg.Path + metadata.FileExt
		if _, err := os.Stat(metadataFile); !errors.Is(err, os.ErrNotExist) {
			continue
		}
		f, err := os.Create(metadataFile)
		if err != nil {
			return err
		}
		err = pkg.Metadata.MarshalJSON(f)
		if err != nil {
			return err
		}
		f.Close()
	}
	return nil
}
