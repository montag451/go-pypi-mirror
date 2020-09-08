package pkg

import (
	"sort"

	"github.com/hashicorp/go-version"
)

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
