package usedtype

import (
	"fmt"
	"go/types"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/tools/go/ssa"
)

type StructDefNodes map[NamedTypeId][]ssa.Value

func (m StructDefNodes) String() string {
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

// FindInPackageDefNodeOfTargetStructType find the SSA nodes that declares global/local variable that are of the same type
// of the targetStructures for each SSA package. These nodes are the "def" nodes in context of SSA.
func FindInPackageDefNodeOfTargetStructType(ssapkgs []*ssa.Package, targetStructs StructMap) StructDefNodes {
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
