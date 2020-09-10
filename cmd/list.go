package cmd

import (
	"flag"
	"fmt"

	"github.com/montag451/go-pypi-mirror/pkg"
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
		name := group.Key.(string)
		if c.name != "" && name != c.name {
			continue
		}
		fmt.Println(name)
		if c.name == "" && c.nameOnly {
			continue
		}
		groups := pkg.GroupByVersion(group.Pkgs)
		for i := len(groups) - 1; i >= 0; i-- {
			fmt.Printf("  %v\n", groups[i].Key)
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
