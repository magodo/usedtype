package usedtype

import (
	"fmt"
	"go/types"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"
)

// The Object.Id() not always guarantees to return a qualified ID for an object.
type NamedTypeId struct {
	Pkg      *packages.Package
	TypeName string
}

func (id NamedTypeId) String() string {
	return fmt.Sprintf("%s (%s)", id.TypeName, id.Pkg.PkgPath)
}

type StructMap map[NamedTypeId]*types.Struct

func (m StructMap) String() string {
	idstrings := []string{}
	ids := []NamedTypeId{}
	i := 0
	for k := range m {
		idstrings = append(idstrings, fmt.Sprintf("%s:%s:%d", k.Pkg.PkgPath, k.TypeName, i))
		ids = append(ids, k)
		i++
	}
	sort.Strings(idstrings)

	output := []string{}
	for _, idstr := range idstrings {
		parts := strings.Split(idstr, ":")
		idx, _ := strconv.Atoi(parts[2])
		id := ids[idx]
		output = append(output, fmt.Sprintf("%s: %s\n", id, m[id]))
	}
	return strings.Join(output, "")
}

type FilterFunc func(epkg *packages.Package, t *types.Struct) bool

func FindExternalPackageStruct(pkgs []*packages.Package, pattern string, filter FilterFunc) StructMap {
	p := regexp.MustCompile(pattern)
	targetStructs := map[NamedTypeId]*types.Struct{}
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
				t, ok := namedType.Underlying().(*types.Struct)
				if !ok {
					continue
				}
				if filter != nil && !filter(epkg, t) {
					continue
				}

				id := NamedTypeId{
					Pkg:      epkg,
					TypeName: obj.Name(),
				}
				targetStructs[id] = t
			}
		}
	}
	return targetStructs
}
