package usedtype

import (
	"fmt"
	"go/types"

	"golang.org/x/tools/go/ssa"
)

type Traversal struct {
	seen seen
}

type seen struct {
	functions    map[*ssa.Function]struct{}
	instructions map[ssa.Instruction]struct{}
	values       map[ssa.Value]struct{}
}

func NewTraversal() Traversal {
	return Traversal{
		seen: seen{
			functions:    map[*ssa.Function]struct{}{},
			instructions: map[ssa.Instruction]struct{}{},
			values:       map[ssa.Value]struct{}{},
		},
	}
}

type WalkInstrCallback func(instr ssa.Instruction)
type WalkValueCallback func(val ssa.Value)

// WalkInPackage traverse inside a usedtype package from all top level functions (skipping other top level members:
// Type, NamedConst and Global). It will iterate each instruction and the value belongs to it.
// Note that only the functions defined in this usedtype package is traversed, it will not cross package boundary.
func (t *Traversal) WalkInPackage(pkg *ssa.Package, icb WalkInstrCallback, vcb WalkValueCallback) {
	t.seen = seen{
		functions:    map[*ssa.Function]struct{}{},
		instructions: map[ssa.Instruction]struct{}{},
		values:       map[ssa.Value]struct{}{},
	}
	for _, m := range pkg.Members {
		switch m := m.(type) {
		case *ssa.NamedConst,
			*ssa.Type:
		case *ssa.Global:
			if _, ok := t.seen.values[m]; ok {
				continue
			}
			t.seen.values[m] = struct{}{}
			if vcb != nil {
				vcb(m)
			}
		case *ssa.Function:
			t.walkFunction(pkg, m, icb, vcb)
		default:
			panic(fmt.Sprintf("unreachable: %T", m))
		}
	}

	// Since the methods of package-level types do not belong to the package "member", which means above member-wise iteration
	// will not cover those methods. We'll handle them below.
	for _, typ := range pkg.Prog.RuntimeTypes() {
		// Only handle package leve types that are named struct
		if !IsUnderlyingNamedStruct(typ) {
			continue
		}
		nt, _ := DereferenceR(typ).(*types.Named)

		// Only handle the current package types
		if nt.Obj().Pkg() != pkg.Pkg {
			continue
		}

		mset := pkg.Prog.MethodSets.MethodSet(typ)
		for i, n := 0, mset.Len(); i < n; i++ {
			t.walkFunction(pkg, pkg.Prog.MethodValue(mset.At(i)), icb, vcb)
		}
	}
}

func (t *Traversal) walkFunction(pkg *ssa.Package, fn *ssa.Function, icb WalkInstrCallback, vcb WalkValueCallback) {
	// We only walk through the functions defined in current package boundary.
	if fn.Package() != nil && fn.Package() != pkg {
		return
	}

	// Record those functions have been traversed, to avoid cyclic call.
	if _, ok := t.seen.functions[fn]; ok {
		return
	}
	t.seen.functions[fn] = struct{}{}

	for _, param := range fn.Params {
		t.walkValue(pkg, param, icb, vcb)
	}

	t.walkInstructions(pkg, fn, icb, vcb)

	for _, anon := range fn.AnonFuncs {
		// functions use anonymous functions defined beneath them
		t.walkFunction(pkg, anon, icb, vcb)
	}
}

func (t *Traversal) walkInstructions(pkg *ssa.Package, fn *ssa.Function, icb WalkInstrCallback, vcb WalkValueCallback) {
	for _, b := range fn.Blocks {
		for _, instr := range b.Instrs {
			if _, ok := t.seen.instructions[instr]; ok {
				continue
			}
			t.seen.instructions[instr] = struct{}{}

			if icb != nil {
				icb(instr)
			}

			// traverse the operands in instructions
			ops := instr.Operands(nil)

			for _, arg := range ops {
				t.walkValue(pkg, *arg, icb, vcb)
			}
		}
	}
}

func (t *Traversal) walkValue(pkg *ssa.Package, v ssa.Value, icb WalkInstrCallback, vcb WalkValueCallback) {
	if v == nil {
		return
	}

	if _, ok := t.seen.values[v]; ok {
		return
	}
	t.seen.values[v] = struct{}{}

	phi, ok := v.(*ssa.Phi)
	if !ok {
		if vcb != nil {
			vcb(v)
		}

		// This is necessary for following the method calls, which are not included
		// in Members of ssa package.
		switch v := v.(type) {
		case *ssa.Function:
			t.walkFunction(pkg, v, icb, vcb)
			return
		}
		return
	}

	applyPhi := func(v *ssa.Phi) {
		for _, e := range v.Edges {
			t.walkValue(pkg, e, icb, vcb)
		}
	}
	applyPhi(phi)
}
