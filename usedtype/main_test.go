package usedtype_test

import (
	"go/types"
	"os"
	"path/filepath"

	"golang.org/x/tools/go/packages"
)

var (
	pathA                 string
	pathInterfaceProperty string
	pathInterfaceRoot     string
	pathInterfaceNest     string
	pathCrossFunc         string
	pathCrossBB           string
	pathCrossFuncNoLink   string
	pathInstrPos          string
)

func init() {
	pwd, _ := os.Getwd()
	pathA = filepath.Join(pwd, "testdata", "src", "a")
	pathInterfaceProperty = filepath.Join(pwd, "testdata", "src", "interface_property")
	pathInterfaceRoot = filepath.Join(pwd, "testdata", "src", "interface_root")
	pathInterfaceNest = filepath.Join(pwd, "testdata", "src", "interface_nest")
	pathCrossFunc = filepath.Join(pwd, "testdata", "src", "cross_func")
	pathCrossBB = filepath.Join(pwd, "testdata", "src", "cross_bb")
	pathCrossFuncNoLink = filepath.Join(pwd, "testdata", "src", "cross_func_no_link")
	pathInstrPos = filepath.Join(pwd, "testdata", "src", "instr_pos")
}

func filterTypeByName(typeName string) func(epkg *packages.Package, t *types.Named) bool {
	return func(epkg *packages.Package, t *types.Named) bool {
		return t.String() == typeName
	}
}

func terraformSchemaTypeFilter(epkg *packages.Package, t *types.Named) bool {
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
			if types.Identical(lastParam.Type(), t) {
				return true
			}
		default:
			continue
		}
	}
	return false
}
