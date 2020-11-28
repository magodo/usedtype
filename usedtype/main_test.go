package usedtype_test

import (
	"go/types"
	"os"
	"path/filepath"

	"golang.org/x/tools/go/packages"
)

var (
	pathA           string
	pathValParam    string
	pathMutateParam string
	pathMultiReturn string
	pathBuildPtrPropInFunctionWithIf string
)

func init() {
	pwd, _ := os.Getwd()
	pathA = filepath.Join(pwd, "testdata", "src", "a")
	pathValParam = filepath.Join(pwd, "testdata", "src", "val_param")
	pathMutateParam = filepath.Join(pwd, "testdata", "src", "mutate_param")
	pathMultiReturn = filepath.Join(pwd, "testdata", "src", "multi_return")
	pathBuildPtrPropInFunctionWithIf = filepath.Join(pwd, "testdata", "src", "build_ptr_prop_in_function_with_if")
}

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
