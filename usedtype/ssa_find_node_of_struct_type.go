package usedtype

import (
	"go/types"
	"golang.org/x/tools/go/ssa"
)

// FindInPackageDefNodeOfTargetStructType find the SSA nodes that declares global/local variable that are of the same type
// of the targetStructures for each SSA package. These nodes are the "def" nodes in context of SSA.
func FindInPackageDefNodeOfTargetStructType(ssapkgs []*ssa.Package, targetStructs StructMap) map[NamedTypeId][]ssa.Value {
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
			pt, ok := vt.(*types.Pointer)
			if !ok {
				return
			}

			// It is possible that defines multiple pointer level for the target type
			for e, ok := pt.Elem().(*types.Pointer); ok; pt = e {}

			nt, ok := pt.Elem().(*types.Named)
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
