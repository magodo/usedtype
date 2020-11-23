package usedtype

import (
	"go/types"
	"golang.org/x/tools/go/ssa"
)

type SSAValue struct {
	Instr ssa.Instruction
	V     ssa.Value
}

// FindInPackageNodeOfTargetStructType find the usedtype nodes that are of the same type of the targetStructures, for each usedtype package.
func FindInPackageNodeOfTargetStructType(ssapkgs []*ssa.Package, targetStructs StructMap) map[NamedTypeId][]SSAValue {
	output := map[NamedTypeId][]SSAValue{}
	for _, pkg := range ssapkgs {
		var cb WalkCallback
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
						SSAValue{
							Instr: instr,
							V:     v,
						})
				}
			}
		}

		ssaTraversal := NewTraversal()
		ssaTraversal.WalkInPackage(pkg, cb)
	}
	return output
}
