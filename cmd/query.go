package cmd

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"text/template"

	"github.com/hashicorp/go-version"
)

type queryCommand struct {
	flags       *flag.FlagSet
	constraints string
	latest      uint
	url         string
	format      string
}

func (c *queryCommand) FlagSet() *flag.FlagSet {
	return c.flags
}

func (c *queryCommand) Execute() error {
	pkgs := c.flags.Args()
	if nbPkgs := len(pkgs); nbPkgs == 0 {
		return errors.New("no package specified")
	} else if nbPkgs > 1 {
		return fmt.Errorf("only one package must be specified, got %v", nbPkgs)
	}
	if c.url == "" {
		return fmt.Errorf("empty URL")
	}
	t, err := template.New("url").Parse(c.url)
	if err != nil {
		return fmt.Errorf("invalid URL template %q: %w", c.url, err)
	}
	var constraints version.Constraints
	if c.constraints != "" {
		var err error
		constraints, err = version.NewConstraint(c.constraints)
		if err != nil {
			return fmt.Errorf("invalid version constraint %q: %w", c.constraints, err)
		}
	}
	var url strings.Builder
	if err := t.Execute(&url, pkgs[0]); err != nil {
		return fmt.Errorf("failed to execute URL template %q: %w", c.url, err)
	}
	resp, err := http.Get(url.String())
	if err != nil {
		return fmt.Errorf("failed to get %q: %w", url.String(), err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to get %q, HTTP code: %v", url.String(), resp.StatusCode)
	}
	var info map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return fmt.Errorf("failed to parse response: %v", err)
	}
	releases, ok := info["releases"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("failed to parse response, missing or invalid key: %q", "releases")
	}
	var versions []*version.Version
	for rawVersion, _ := range releases {
		version, err := version.NewVersion(rawVersion)
		if err != nil {
			fmt.Errorf("unable to parse version %q: %w", rawVersion, err)
		}
		if c.constraints == "" || constraints.Check(version) {
			versions = append(versions, version)
		}
	}
	sort.Sort(version.Collection(versions))
	if c.latest > 0 {
		versions = versions[len(versions)-int(c.latest):]
	}
	switch c.format {
	case "", "oneline":
		for i := len(versions) - 1; i >= 0; i-- {
			fmt.Println(versions[i])
		}
	case "json":
		reversedVersions := make([]string, 0, len(versions))
		for i := len(versions) - 1; i >= 0; i-- {
			reversedVersions = append(reversedVersions, versions[i].Original())
		}
		json.NewEncoder(os.Stdout).Encode(reversedVersions)
	}
	return nil
}

func init() {
	cmd := queryCommand{}
	flags := flag.NewFlagSet("query", flag.ExitOnError)
	flags.StringVar(&cmd.constraints, "constraints", "", "version constraints")
	flags.UintVar(&cmd.latest, "latest", 0, "list only the latest `N` versions")
	flags.StringVar(&cmd.url, "url", "https://pypi.org/pypi/{{ . }}/json", "index URL template")
	flags.StringVar(&cmd.format, "format", "oneline", "output format (oneline or json)")
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [options] PKG\n", flags.Name())
		fmt.Fprintln(flags.Output(), "Options:")
		flags.PrintDefaults()
	}
	cmd.flags = flags
	registerCommand(&cmd)
}
