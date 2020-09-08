package pkg

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/montag451/go-pypi-mirror/metadata"

	"github.com/hashicorp/go-version"
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

type pkgSorter struct {
	pkgs []*Pkg
	by   func(p1, p2 *Pkg) bool
}

func (s *pkgSorter) Len() int {
	return len(s.pkgs)
}

func (s *pkgSorter) Less(i, j int) bool {
	return s.by(s.pkgs[i], s.pkgs[j])
}

func (s *pkgSorter) Swap(i, j int) {
	s.pkgs[i], s.pkgs[j] = s.pkgs[j], s.pkgs[i]
}

type sortBy func(p1, p2 *Pkg) bool

func (by sortBy) sort(pkgs []*Pkg) {
	sorter := &pkgSorter{
		pkgs: pkgs,
		by:   by,
	}
	sort.Sort(sorter)
}

func SortByVersion(pkgs []*Pkg, desc bool) {
	version := func(p1, p2 *Pkg) bool {
		v1, err1 := version.NewVersion(p1.Metadata.Version)
		v2, err2 := version.NewVersion(p2.Metadata.Version)
		if err1 != nil || err2 != nil {
			if desc {
				return p1.Metadata.Version > p2.Metadata.Version
			} else {
				return p1.Metadata.Version < p2.Metadata.Version
			}
		}
		if desc {
			return v1.Compare(v2) == 1
		} else {
			return v1.Compare(v2) == -1
		}
	}
	sortBy(version).sort(pkgs)
}
