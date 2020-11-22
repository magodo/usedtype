package main

import (
	"fmt"
	myssa "github.com/magodo/usedtype/ssa"
	"go/types"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
)

const usage = `usedtype <package>`

func main() {
	pkgs, ssapkgs := buildPackages()

	// Analyze all the target external packages and get a list of types.Object
	pattern := "^sdk$"
	targetStructs := locateExternalPackageStruct(pkgs, pattern, terraformSchemaTypeFilter)
	//fmt.Println(targetStructs)

	// Explore the packages under test to see whether there is ssa node whose type matches any target struct.
	// For each match, we will walk the dominator tree from that node in backward, to record the usage of each
	// field of the struct.
	output := findInPackageNodeOfTargetStructType(ssapkgs, targetStructs)
	fmt.Println(output)

	// Now we need to recursively backward analyze from each found node, to record all the field accesses.
	for _, nodes := range output {
		for _, node := range nodes {
			var branches myssa.UseDefBranches
			branches = []myssa.UseDefBranch{
				myssa.NewUseDefBranch(node.instr, node.v),
			}
			newbranches := branches.Walk()
			for _, b := range newbranches {
				fmt.Println(b)
			}
		}
	}
	return
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

type ssaValue struct {
	instr ssa.Instruction
	v     ssa.Value
}

// findInPackageNodeOfTargetStructType find the ssa nodes that are of the same type of the targetStructures, for each ssa package.
func findInPackageNodeOfTargetStructType(ssapkgs []*ssa.Package, targetStructs structMap) map[namedTypeId][]ssaValue {
	output := map[namedTypeId][]ssaValue{}
	for _, pkg := range ssapkgs {
		var cb myssa.WalkCallback
		cb = func(instr ssa.Instruction, v ssa.Value) {
			vt := v.Type()
			nt, ok := vt.(*types.Named)
			if !ok {
				return
			}
			st, ok := nt.Underlying().(*types.Struct)
			if !ok {
				return
			}
			for tid, tv := range targetStructs {
				if types.Identical(tv, st) {
					output[tid] = append(output[tid],
						ssaValue{
							instr: instr,
							v:     v,
						})
				}
			}
		}
		myssa.WalkInPackage(pkg, cb)
	}
	return output
}
