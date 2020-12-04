package usedtype

import (
	"go/types"
	"regexp"

	"golang.org/x/tools/go/packages"
)

type NamedTypeSet map[*types.Named]struct{}

type FilterFunc func(epkg *packages.Package, t *types.Named) bool

// Find among the external depended packages of "pkgs", whose import path matching "pattern", all the
// named types. If "filter" is non-nil, it is used to further filter the named types.
func FindExternalPackageNamedType(pkgs []*packages.Package, pattern string, filter FilterFunc) NamedTypeSet {
	p := regexp.MustCompile(pattern)
	tset := map[*types.Named]struct{}{}
	for _, pkg := range pkgs {
		for epkgImportPath, epkg := range pkg.Imports {
			if !p.MatchString(epkgImportPath) {
				continue
			}
			for _, obj := range epkg.TypesInfo.Defs {
				if _, ok := obj.(*types.TypeName); !ok {
					continue
				}
				namedType, ok := obj.Type().(*types.Named)
				if !ok {
					continue
				}
				if filter != nil && !filter(epkg, namedType) {
					continue
				}

				tset[namedType] = struct{}{}
			}
		}
	}
	return tset
}
