package cmd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"

	"github.com/montag451/go-pypi-mirror/internal/flagutil"
	"github.com/montag451/go-pypi-mirror/pkg"
)

type downloadCommand struct {
	flags            *flag.FlagSet
	requirements     flagutil.StringSlice
	dest             string
	indexUrl         string
	proxy            string
	allowBinary      bool
	platform         flagutil.StringSlice
	pythonVersion    string
	implementation   string
	noBuildIsolation bool
	abi              flagutil.StringSlice
	pip              string
}

func (c *downloadCommand) FlagSet() *flag.FlagSet {
	return c.flags
}

func (c *downloadCommand) Execute(context.Context) error {
	pkgs := c.FlagSet().Args()
	if len(pkgs) == 0 && len(c.requirements) == 0 {
		return errors.New("at least one requirements file or package must be specified")
	}
	args := make([]string, 0, 3+len(pkgs)+2*len(c.requirements))
	args = append(args, "download", "-d", c.dest)
	if c.indexUrl != "" {
		args = append(args, "--index-url", c.indexUrl)
	}
	if c.proxy != "" {
		args = append(args, "--proxy", c.proxy)
	}
	if !c.allowBinary {
		args = append(args, "--no-binary", ":all:")
	}
	if len(c.platform) > 0 || c.pythonVersion != "" || c.implementation != "" || len(c.abi) > 0 {
		args = append(args, "--only-binary", ":all:")
	}
	for _, p := range c.platform {
		args = append(args, "--platform", p)
	}
	if c.pythonVersion != "" {
		args = append(args, "--python-version", c.pythonVersion)
	}
	if c.implementation != "" {
		args = append(args, "--implementation", c.implementation)
	}
	if c.noBuildIsolation {
		args = append(args, "--no-build-isolation")
	}
	for _, a := range c.abi {
		args = append(args, "--abi", a)
	}
	for _, r := range c.requirements {
		args = append(args, "-r", r)
	}
	args = append(args, pkgs...)
	cmd := exec.Command(c.pip, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failure while executing %q: %w", cmd, err)
	}
	return pkg.CreateMetadataFiles(c.dest, false)
}

func init() {
	cmd := downloadCommand{
		requirements: make([]string, 0),
	}
	flags := flag.NewFlagSet("download", flag.ExitOnError)
	flags.Var(&cmd.requirements, "requirements", "requirements file")
	flags.StringVar(&cmd.dest, "download-dir", ".", "download directory")
	flags.StringVar(&cmd.indexUrl, "index-url", "", "index URL")
	flags.StringVar(&cmd.proxy, "proxy", "", "proxy address in the form [user:passwd@]proxy.server:port")
	flags.BoolVar(&cmd.allowBinary, "allow-binary", false, "allow binary")
	flags.Var(&cmd.platform, "platform", "platform")
	flags.StringVar(&cmd.pythonVersion, "python-version", "", "Python version")
	flags.StringVar(&cmd.implementation, "implementation", "", "implementation")
	flags.Var(&cmd.abi, "abi", "Python ABI")
	flags.BoolVar(&cmd.noBuildIsolation, "no-build-isolation", false, "disable isolation when building")
	flags.StringVar(&cmd.pip, "pip", "pip3", "pip executable")
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [options] [pkgs]\n", flags.Name())
		fmt.Fprintln(flags.Output(), "Options:")
		flags.PrintDefaults()
	}
	cmd.flags = flags
	RegisterCommand(&cmd)
}
