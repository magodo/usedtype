package usedtype

import (
	"fmt"
	"go/token"
	"go/types"
	"golang.org/x/tools/go/packages"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/tools/go/ssa"
)

type StructDefValues map[NamedTypeId][]ssa.Value

type SSAValue struct {
	Value ssa.Value
	Fset  *token.FileSet
}

type SSAValues []SSAValue

func (m StructDefValues) String() string {
	idstrings := []string{}
	ids := []NamedTypeId{}
	i := 0
	for k := range m {
		idstrings = append(idstrings, fmt.Sprintf("%s:%s:%d", k.Pkg.PkgPath, k.TypeName, i))
		ids = append(ids, k)
		i++
	}
	sort.Strings(idstrings)

	outputs := []string{}
	for _, idstr := range idstrings {
		parts := strings.Split(idstr, ":")
		idx, _ := strconv.Atoi(parts[2])
		id := ids[idx]
		output := id.String() + "\n"
		for _, v := range m[id] {
			output += fmt.Sprintf("\t%s\n", id.Pkg.Fset.Position(v.Pos()))
		}
		outputs = append(outputs, output)
	}
	return strings.Join(outputs, "")
}

func (values SSAValues) String() string {
	var valueStrs []string

	for _, value := range values {
		valueStrs = append(valueStrs, fmt.Sprintf("%s: %s\n", value.Fset.Position(value.Value.Pos()), value.Value))
	}
	sort.Strings(valueStrs)
	return strings.Join(valueStrs, "")
}

// FindInPackageDefValueOfTargetStructType find the SSA nodes that declares global/local variable that are of the same type
// of the targetStructures for each SSA package. These nodes are the "def" nodes in context of SSA.
func FindInPackageDefValueOfTargetStructType(ssapkgs []*ssa.Package, targetStructs StructMap) StructDefValues {
	output := map[NamedTypeId][]ssa.Value{}
	for _, pkg := range ssapkgs {
		var cb WalkCallback
		cb = func(v ssa.Value) {
			switch v.(type) {
			// Local variable declaration in functions or global variable declaration
			case *ssa.Alloc,
				*ssa.Global:
				// continue
			default:
				return
			}

			// Since both local and global variable in SSA is a reserved memory for the target type, so the node
			// type is always a pointer.
			vt := v.Type()

			for pt, ok := vt.(*types.Pointer); ok; {
				vt = pt.Elem()
				pt, ok = vt.(*types.Pointer)
			}

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
					output[tid] = append(output[tid], v)
				}
			}
		}

		ssaTraversal := NewTraversal()
		ssaTraversal.WalkInPackage(pkg, cb)
	}
	return output
}

func FindInPackageAllDefValue(pkgs []*packages.Package, ssapkgs []*ssa.Package) SSAValues {
	output := []SSAValue{}
	for i := range ssapkgs {
		ssapkg := ssapkgs[i]
		pkg := pkgs[i]
		var cb WalkCallback
		cb = func(v ssa.Value) {
			switch v := v.(type) {
			case *ssa.Alloc,
				*ssa.Parameter:
				// continue
			case *ssa.Global:
				// E.g. init$guard
				if v.Object() == nil {
					return
				}
			default:
				return
			}
			output = append(output, SSAValue{
				Value: v,
				Fset:  pkg.Fset,
			})
		}
		ssaTraversal := NewTraversal()
		ssaTraversal.WalkInPackage(ssapkg, cb)
	}

	return output
}
