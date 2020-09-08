package pkg

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/montag451/go-pypi-mirror/metadata"
)

type Pkg struct {
	Path     string
	Filename string
	Metadata *metadata.Metadata
}

func newPackage(path string) (*Pkg, error) {
	meta, err := metadata.Get(path)
	if err != nil {
		return nil, fmt.Errorf("error while processing %q: %w", path, err)
	}
	return &Pkg{path, filepath.Base(path), meta}, nil
}

func GroupByNormName(pkgs []*Pkg) map[string][]*Pkg {
	byNormName := make(map[string][]*Pkg)
	for _, pkg := range pkgs {
		normName := pkg.Metadata.NormName
		byNormName[normName] = append(byNormName[normName], pkg)
	}
	return byNormName
}

func List(dir string, fixNames bool) ([]*Pkg, error) {
	pkgs := make([]*Pkg, 0)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && !strings.HasSuffix(path, metadata.FileExt) {
			p, err := newPackage(path)
			if err != nil {
				return err
			}
			pkgs = append(pkgs, p)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if fixNames {
		for _, pkgs := range GroupByNormName(pkgs) {
			FixNames(pkgs)
		}
	}
	return pkgs, nil
}

// TODO: sort respecting locale
func ListByNames(dir string) (map[string][]*Pkg, error) {
	pkgs, err := List(dir, true)
	if err != nil {
		return nil, err
	}
	byName := make(map[string][]*Pkg)
	for _, pkg := range pkgs {
		name := pkg.Metadata.Name
		byName[name] = append(byName[name], pkg)
	}
	return byName, nil
}

func ListNames(dir string) ([]string, error) {
	byName, err := ListByNames(dir)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(byName))
	for name, _ := range byName {
		names = append(names, name)
	}
	return names, nil
}

func FixNames(pkgs []*Pkg) {
	byTrust := make(map[bool][]*Pkg, 2)
	for _, pkg := range pkgs {
		trusted := pkg.Metadata.Trusted
		byTrust[trusted] = append(byTrust[trusted], pkg)
	}
	trustedPkgs := byTrust[true]
	if len(trustedPkgs) > 0 {
		trustedName := trustedPkgs[0].Metadata.Name
		for _, pkg := range byTrust[false] {
			pkg.Metadata.Name = trustedName
		}
	}
}
