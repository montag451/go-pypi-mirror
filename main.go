package main

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"

	"github.com/montag451/go-pypi-mirror/cmd"
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

func createMirror(downloadDir string, mirrorDir string, pkgs []*pkg.Pkg) error {
	downloadDir, err := filepath.Abs(downloadDir)
	if err != nil {
		return err
	}
	mirrorDir, err = filepath.Abs(mirrorDir)
	if err != nil {
		return err
	}
	if pkgs == nil || len(pkgs) == 0 {
		var err error
		pkgs, err = pkg.List(downloadDir, false)
		if err != nil {
			return err
		}
	}
	pkgsByNormName := pkg.GroupByNormName(pkgs)
	rootPkgs := make([]*pkg.Pkg, 0, len(pkgsByNormName))
	for normName, pkgs := range pkgsByNormName {
		fmt.Println(normName)
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
	f, err := os.Create(filepath.Join(mirrorDir, "index.html"))
	if err != nil {
		return err
	}
	defer f.Close()
	return generateRootHTML(f, rootPkgs)
}

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
	}
}
