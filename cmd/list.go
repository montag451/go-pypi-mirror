package cmd

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/montag451/go-pypi-mirror/pkg"
)

type listCommand struct {
	flags       *flag.FlagSet
	downloadDir string
	nameOnly    bool
	name        string
	json        bool
	useNormName bool
}

func (c *listCommand) FlagSet() *flag.FlagSet {
	return c.flags
}

func (c *listCommand) Execute(context.Context) error {
	pkgs, err := pkg.List(c.downloadDir, true)
	if err != nil {
		return err
	}
	groups := pkg.GroupByName(pkgs)
	pkgsByName := make([]map[string]interface{}, len(groups))
	for i, group := range groups {
		name := group.Key.(string)
		if c.useNormName {
			name = group.Pkgs[0].Metadata.NormName
		}
		if c.name != "" && name != c.name {
			continue
		}
		groups := pkg.GroupByVersion(group.Pkgs)
		versions := make([]string, len(groups))
		for i := len(groups) - 1; i >= 0; i-- {
			versions[i] = groups[i].Key.(string)
		}
		pkgsByName[i] = map[string]interface{}{
			"name":     name,
			"versions": versions,
		}
	}
	if c.json {
		return json.NewEncoder(os.Stdout).Encode(pkgsByName)
	}
	for _, pkg := range pkgsByName {
		fmt.Println(pkg["name"])
		if c.name == "" && c.nameOnly {
			continue
		}
		for _, v := range pkg["versions"].([]string) {
			fmt.Printf("  %s\n", v)
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
	flags.BoolVar(&cmd.json, "json", false, "JSON output")
	flags.BoolVar(&cmd.useNormName, "use-norm-name", false, "use the normalized name instead of the regular name")
	cmd.flags = flags
	RegisterCommand(&cmd)
}
