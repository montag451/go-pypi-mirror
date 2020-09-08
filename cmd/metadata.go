package cmd

import (
	"errors"
	"flag"
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

type writeMetadataCommand struct {
	flags       *flag.FlagSet
	downloadDir string
	overwrite   bool
}

func (c *writeMetadataCommand) FlagSet() *flag.FlagSet {
	return c.flags
}

func (c *writeMetadataCommand) Execute() error {
	return createMetadataFiles(c.downloadDir, c.overwrite)
}

func init() {
	cmd := writeMetadataCommand{}
	flags := flag.NewFlagSet("write-metadata", flag.ContinueOnError)
	flags.StringVar(&cmd.downloadDir, "download-dir", "", "download dir")
	flags.BoolVar(&cmd.overwrite, "overwrite", false, "overwrite metadata files")
	cmd.flags = flags
	registerCommand(&cmd)
}
