package pkg

import (
	"sort"

	"github.com/hashicorp/go-version"

	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

func SortByVersion(pkgs []*Pkg, desc bool) {
	version := func(i, j int) bool {
		p1, p2 := pkgs[i], pkgs[j]
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
	sort.Slice(pkgs, version)
}

func SortByNormName(pkgs []*Pkg, desc bool) {
	collator := collate.New(language.MustParse("en-US"))
	normName := func(i, j int) bool {
		p1, p2 := pkgs[i], pkgs[j]
		n1 := p1.Metadata.NormName
		n2 := p2.Metadata.NormName
		if desc {
			return collator.CompareString(n1, n2) == 1
		} else {
			return collator.CompareString(n1, n2) == -1
		}
	}
	sort.Slice(pkgs, normName)
}

func SortByName(pkgs []*Pkg, desc bool) {
	collator := collate.New(language.MustParse("en-US"))
	name := func(i, j int) bool {
		p1, p2 := pkgs[i], pkgs[j]
		n1 := p1.Metadata.Name
		n2 := p2.Metadata.Name
		if desc {
			return collator.CompareString(n1, n2) == 1
		} else {
			return collator.CompareString(n1, n2) == -1
		}
	}
	sort.Slice(pkgs, name)
}

type Group struct {
	Key  interface{}
	Pkgs []*Pkg
}

type sortFunc func(pkgs []*Pkg, desc bool)
type keyFunc func(pkg *Pkg) interface{}

func GroupBy(pkgs []*Pkg, sort sortFunc, key keyFunc) []*Group {
	groups := make([]*Group, 0)
	if len(pkgs) == 0 {
		return groups
	}
	sort(pkgs, false)
	currentKey := key(pkgs[0])
	first := 0
	for i, pkg := range pkgs {
		k := key(pkg)
		if currentKey != k {
			group := &Group{currentKey, pkgs[first:i]}
			groups = append(groups, group)
			currentKey = k
			first = i
		}
	}
	group := &Group{currentKey, pkgs[first:]}
	return append(groups, group)
}

func GroupByVersion(pkgs []*Pkg) []*Group {
	key := func(pkg *Pkg) interface{} {
		return pkg.Metadata.Version
	}
	return GroupBy(pkgs, SortByVersion, key)
}

func GroupByNormName(pkgs []*Pkg) []*Group {
	key := func(pkg *Pkg) interface{} {
		return pkg.Metadata.NormName
	}
	return GroupBy(pkgs, SortByNormName, key)
}

func GroupByName(pkgs []*Pkg) []*Group {
	key := func(pkg *Pkg) interface{} {
		return pkg.Metadata.Name
	}
	return GroupBy(pkgs, SortByName, key)
}
