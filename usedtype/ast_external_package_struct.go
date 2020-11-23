package usedtype

import (
	"go/types"
	"golang.org/x/tools/go/packages"
	"regexp"
)

// The Object.Id() not always guarantees to return a qualified ID for an object.
type NamedTypeId struct {
	pkg  *types.Package
	name string
}

type StructMap map[NamedTypeId]*types.Struct

type FilterFunc func(epkg *packages.Package, t *types.Struct) bool

func FindExternalPackageStruct(pkgs []*packages.Package, pattern string, filter FilterFunc) StructMap {
	p := regexp.MustCompile(pattern)
	targetStructs := map[NamedTypeId]*types.Struct{}
	for _, pkg := range pkgs {
		for epkgName, epkg := range pkg.Imports {
			if !p.MatchString(epkgName) {
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
				t, ok := namedType.Underlying().(*types.Struct)
				if !ok {
					continue
				}
				if filter != nil && !filter(epkg, t) {
					continue
				}

				id := NamedTypeId{
					pkg:  obj.Pkg(),
					name: obj.Name(),
				}
				targetStructs[id] = t
			}
		}
	}
	return targetStructs
}
