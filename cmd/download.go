package cmd

import (
	"flag"
	"fmt"
	"os"
	"os/exec"

	"github.com/montag451/go-pypi-mirror/pkg"
)

type requirementsValue []string

func (r *requirementsValue) String() string {
	return fmt.Sprint(*r)
}

func (r *requirementsValue) Set(s string) error {
	*r = append(*r, s)
	return nil
}

type downloadCommand struct {
	flags          *flag.FlagSet
	pkgs           []string
	requirements   requirementsValue
	dest           string
	indexUrl       string
	allowBinary    bool
	platform       string
	pythonVersion  string
	implementation string
	abi            string
	pip            string
}

func (c *downloadCommand) FlagSet() *flag.FlagSet {
	return c.flags
}

func (c *downloadCommand) Execute() error {
	c.pkgs = c.FlagSet().Args()
	if len(c.pkgs) == 0 && len(c.requirements) == 0 {
		return nil
	}
	args := make([]string, 0, 3+len(c.pkgs)+2*len(c.requirements))
	args = append(args, "download", "-d", c.dest)
	if c.indexUrl != "" {
		args = append(args, "--index-url", c.indexUrl)
	}
	if !c.allowBinary {
		args = append(args, "--no-binary", ":all:")
	}
	if c.platform != "" || c.pythonVersion != "" || c.implementation != "" || c.abi != "" {
		args = append(args, "--only-binary", ":all:")
	}
	if c.platform != "" {
		args = append(args, "--platform", c.platform)
	}
	if c.pythonVersion != "" {
		args = append(args, "--python-version", c.pythonVersion)
	}
	if c.implementation != "" {
		args = append(args, "--implementation", c.implementation)
	}
	if c.abi != "" {
		args = append(args, "--abi", c.abi)
	}
	for _, r := range c.requirements {
		args = append(args, "-r", r)
	}
	args = append(args, c.pkgs...)
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
	flags := flag.NewFlagSet("download", flag.ContinueOnError)
	flags.Var(&cmd.requirements, "requirements", "requirements file")
	flags.StringVar(&cmd.dest, "download-dir", ".", "download directory")
	flags.StringVar(&cmd.indexUrl, "index-url", "", "index URL")
	flags.BoolVar(&cmd.allowBinary, "allow-binary", false, "allow binary")
	flags.StringVar(&cmd.platform, "platform", "", "platform")
	flags.StringVar(&cmd.pythonVersion, "python-version", "", "Python version")
	flags.StringVar(&cmd.implementation, "implementation", "", "implementation")
	flags.StringVar(&cmd.abi, "abi", "", "Python ABI")
	flags.StringVar(&cmd.pip, "pip", "pip3", "pip executable")
	cmd.flags = flags
	registerCommand(&cmd)
}
