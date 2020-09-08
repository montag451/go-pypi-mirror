package pkg

import (
	"os"
	"path/filepath"
	"strings"
)

func List(dir string, fixNames bool) ([]*Pkg, error) {
	pkgs := make([]*Pkg, 0)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && !strings.HasSuffix(path, metadataExt) {
			p, err := New(path)
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
		for _, group := range GroupByNormName(pkgs) {
			FixNames(group.Pkgs)
		}
	}
	return pkgs, nil
}

func ListNames(dir string) ([]string, error) {
	pkgs, err := List(dir, true)
	if err != nil {
		return nil, err
	}
	groups := GroupByName(pkgs)
	names := make([]string, 0, len(groups))
	for _, group := range groups {
		names = append(names, group.Key.(string))
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
