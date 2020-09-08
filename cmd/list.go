package cmd

import (
	"flag"
	"fmt"

	"github.com/montag451/go-pypi-mirror/pkg"

	orderedmap "github.com/wk8/go-ordered-map"
)

type listCommand struct {
	flags       *flag.FlagSet
	downloadDir string
	nameOnly    bool
	name        string
}

func (c *listCommand) FlagSet() *flag.FlagSet {
	return c.flags
}

func (c *listCommand) Execute() error {
	pkgs, err := pkg.List(c.downloadDir, true)
	if err != nil {
		return err
	}
	groups := pkg.GroupByName(pkgs)
	for _, group := range groups {
		name := group.Key
		if c.name != "" && name != c.name {
			continue
		}
		fmt.Println(name)
		if c.name == "" && c.nameOnly {
			continue
		}
		pkgs := group.Pkgs
		pkg.SortByVersion(pkgs, true)
		versions := orderedmap.New()
		for _, pkg := range pkgs {
			versions.Set(pkg.Metadata.Version, true)
		}
		for pair := versions.Oldest(); pair != nil; pair = pair.Next() {
			fmt.Printf("  %v\n", pair.Key)
		}
	}
	return nil
}

func init() {
	cmd := listCommand{}
	flags := flag.NewFlagSet("list", flag.ExitOnError)
	flags.StringVar(&cmd.downloadDir, "download-dir", ".", "download dir")
	flags.BoolVar(&cmd.nameOnly, "name-only", false, "list only the names of the packages")
	flags.StringVar(&cmd.name, "name", "", "list only the versions of `name`")
	cmd.flags = flags
	registerCommand(&cmd)
}
