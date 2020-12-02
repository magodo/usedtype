package usedtype

import (
	"go/types"
	"golang.org/x/tools/go/packages"
	"regexp"
)

type StructSet map[*types.Named]struct{}

type FilterFunc func(epkg *packages.Package, t *types.Named) bool

func FindExternalPackageStruct(pkgs []*packages.Package, pattern string, filter FilterFunc) StructSet {
	p := regexp.MustCompile(pattern)
	tset:= map[*types.Named]struct{}{}
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
				if ! IsUnderlyingNamedStruct(namedType) {
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
