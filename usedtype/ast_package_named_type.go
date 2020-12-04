package usedtype

import (
	"go/types"
	"regexp"

	"golang.org/x/tools/go/packages"
)

type NamedTypeSet map[*types.Named]struct{}

type FilterFunc func(epkg *packages.Package, t *types.Named) bool

// Find among the packages of "pkgs", whose import path matching "pattern", all the
// Named types. If "filter" is non-nil, it is used to further filter the Named types.
func FindPackageNamedType(pkgs []*packages.Package, pattern string, filter FilterFunc) NamedTypeSet {
	p := regexp.MustCompile(pattern)
	tset := map[*types.Named]struct{}{}
	pkgSet := map[*packages.Package]struct{}{}
	for _, pkg := range pkgs {
		pkgSet[pkg] = struct{}{}
		for _, epkg := range pkg.Imports {
			pkgSet[epkg] = struct{}{}
		}
	}
	for pkg := range pkgSet {
		if !p.MatchString(pkg.PkgPath) {
			continue
		}
		for _, obj := range pkg.TypesInfo.Defs {
			if _, ok := obj.(*types.TypeName); !ok {
				continue
			}
			namedType, ok := obj.Type().(*types.Named)
			if !ok {
				continue
			}
			if filter != nil && !filter(pkg, namedType) {
				continue
			}

			tset[namedType] = struct{}{}
		}
	}
	return tset
}
