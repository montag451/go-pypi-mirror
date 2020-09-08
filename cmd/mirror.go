package cmd

import (
	"errors"
	"flag"
	"html/template"
	"io"
	"os"
	"path/filepath"

	"github.com/montag451/go-pypi-mirror/pkg"
)

var (
	rootHTMLTemplate = template.Must(template.New("root").Parse(`
<!DOCTYPE html>
<html>
  <head>
    <title>Simple index</title>
  </head>
  <body>
    {{- range . }}
    <a href="{{ .Metadata.NormName }}">{{ .Metadata.Name }}</a>
    {{- end }}
  </body>
</html>
`))
	packageHTMLTemplate = template.Must(template.New("pkg").Parse(`
{{- $firstPkg := index . 0 }}
<!DOCTYPE html>
<html>
  <head>
    <title>Links for {{ $firstPkg.Metadata.Name }}</title>
  </head>
  <body>
    <h1>Links for {{ $firstPkg.Metadata.Name }}</h1>
    {{- range . }}
    <a href="{{ .Filename }}#sha256={{ .Metadata.Hash }}">{{ .Filename }}</a><br/>
    {{- end }}
  </body>
</html>
`))
)

func generateRootHTML(w io.Writer, pkgs []*pkg.Pkg) error {
	return rootHTMLTemplate.ExecuteTemplate(w, "root", pkgs)
}

func generatePackageHTML(w io.Writer, pkgs []*pkg.Pkg) error {
	return packageHTMLTemplate.ExecuteTemplate(w, "pkg", pkgs)
}

type createCommand struct {
	flags       *flag.FlagSet
	downloadDir string
	mirrorDir   string
}

func (c *createCommand) FlagSet() *flag.FlagSet {
	return c.flags
}

func (c *createCommand) Execute() error {
	downloadDir, err := filepath.Abs(c.downloadDir)
	if err != nil {
		return err
	}
	mirrorDir, err := filepath.Abs(c.mirrorDir)
	if err != nil {
		return err
	}
	pkgs, err := pkg.List(downloadDir, false)
	if err != nil {
		return err
	}
	pkgsByNormName := pkg.GroupByNormName(pkgs)
	rootPkgs := make([]*pkg.Pkg, 0, len(pkgsByNormName))
	for normName, pkgs := range pkgsByNormName {
		dir := filepath.Join(mirrorDir, normName)
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
		pkg.FixNames(pkgs)
		for _, pkg := range pkgs {
			dest := filepath.Join(dir, pkg.Filename)
			link, err := filepath.Rel(dir, pkg.Path)
			if err != nil {
				return err
			}
			err = os.Symlink(link, dest)
			if err != nil && !errors.Is(err, os.ErrExist) {
				return err
			}
		}
		f, err := os.Create(filepath.Join(dir, "index.html"))
		if err != nil {
			return err
		}
		err = generatePackageHTML(f, pkgs)
		f.Close()
		if err != nil {
			return err
		}
		rootPkgs = append(rootPkgs, pkgs[0])
	}
	if len(rootPkgs) > 0 {
		f, err := os.Create(filepath.Join(mirrorDir, "index.html"))
		if err != nil {
			return err
		}
		defer f.Close()
		return generateRootHTML(f, rootPkgs)
	}
	return nil
}

func init() {
	cmd := createCommand{}
	flags := flag.NewFlagSet("create", flag.ContinueOnError)
	flags.StringVar(&cmd.downloadDir, "download-dir", ".", "download dir")
	flags.StringVar(&cmd.mirrorDir, "mirror-dir", ".", "mirror dir")
	cmd.flags = flags
	registerCommand(&cmd)
}
