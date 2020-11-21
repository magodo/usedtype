package main

import (
	"go/types"
	"golang.org/x/tools/go/packages"
	"regexp"
)

// The Object.Id() not always guarantees to return a qualified ID for an object.
type namedTypeId struct {
	pkg  *types.Package
	name string
}

type structMap map[namedTypeId]*types.Struct

type filterFunc func(epkg *packages.Package, t *types.Struct) bool

func terraformSchemaTypeFilter(epkg *packages.Package, t *types.Struct) bool {
	scope := epkg.Types.Scope()
	for _, topType := range scope.Names() {
		et := scope.Lookup(topType).Type()
		switch et := et.(type) {
		case *types.Named:
			var c, d *types.Func
			for i := 0; i < et.NumMethods(); i++ {
				m := et.Method(i)
				switch m.Name() {
				case "CreateOrUpdate",
					"Create":
					c = m
				case "Delete":
					d = m
				}
			}
			// Terraform only care resources that can be created and deleted.
			if c == nil || d == nil {
				continue
			}
			signature := c.Type().(*types.Signature)
			lastParam := signature.Params().At(signature.Params().Len() - 1)
			nt, ok := lastParam.Type().(*types.Named)
			if !ok {
				continue
			}
			st, ok := nt.Underlying().(*types.Struct)
			if !ok {
				continue
			}
			if types.Identical(st, t) {
				return true
			}
		default:
			continue
		}
	}
	return false
}

func locateExternalPackageStruct(pkgs []*packages.Package, pattern string, filter filterFunc) structMap {
	p := regexp.MustCompile(pattern)
	targetStructs := map[namedTypeId]*types.Struct{}
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

				id := namedTypeId{
					pkg:  obj.Pkg(),
					name: obj.Name(),
				}
				targetStructs[id] = t
			}
		}
	}
	return targetStructs
}
