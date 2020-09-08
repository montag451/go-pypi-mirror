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
		for _, pkgs := range ListByNormName(pkgs) {
			FixNames(pkgs)
		}
	}
	return pkgs, nil
}

func ListByNormName(pkgs []*Pkg) map[string][]*Pkg {
	byNormName := make(map[string][]*Pkg)
	for _, pkg := range pkgs {
		normName := pkg.Metadata.NormName
		byNormName[normName] = append(byNormName[normName], pkg)
	}
	return byNormName
}

func ListByName(dir string) (map[string][]*Pkg, error) {
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
	byName, err := ListByName(dir)
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
