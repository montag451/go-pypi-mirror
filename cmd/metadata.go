package cmd

import (
	"context"
	"flag"

	"github.com/montag451/go-pypi-mirror/pkg"
)

type writeMetadataCommand struct {
	flags       *flag.FlagSet
	downloadDir string
	overwrite   bool
}

func (c *writeMetadataCommand) FlagSet() *flag.FlagSet {
	return c.flags
}

func (c *writeMetadataCommand) Execute(context.Context) error {
	return pkg.CreateMetadataFiles(c.downloadDir, c.overwrite)
}

func init() {
	cmd := writeMetadataCommand{}
	flags := flag.NewFlagSet("write-metadata", flag.ExitOnError)
	flags.StringVar(&cmd.downloadDir, "download-dir", "", "download dir")
	flags.BoolVar(&cmd.overwrite, "overwrite", false, "overwrite metadata files")
	cmd.flags = flags
	registerCommand(&cmd)
}
