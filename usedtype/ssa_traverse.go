package usedtype

import (
	"fmt"
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

type WalkCallback func(val ssa.Value)

// WalkInPackage traverse inside a usedtype package from all top level functions (skipping other top level members:
// Type, NamedConst and Global). It will iterate each instruction and the value belongs to it.
// Note that only the functions defined in this usedtype package is traversed, it will not cross package boundary.
func (t *Traversal) WalkInPackage(pkg *ssa.Package, cb WalkCallback) {
	t.seen = seen{
		functions:    map[*ssa.Function]struct{}{},
		instructions: map[ssa.Instruction]struct{}{},
		values:       map[ssa.Value]struct{}{},
	}
	for _, m := range pkg.Members {
		switch m := m.(type) {
		case *ssa.Type,
			*ssa.NamedConst:
			// nothing to do, since it will not appear any Value of target type
		case *ssa.Global:
			if _, ok := t.seen.values[m]; ok {
				return
			}
			t.seen.values[m] = struct{}{}
			cb(m)
		case *ssa.Function:
			t.walkFunction(pkg, m, cb)
		default:
			panic(fmt.Sprintf("unreachable: %T", m))
		}
	}
}

func (t *Traversal) walkFunction(pkg *ssa.Package, fn *ssa.Function, cb WalkCallback) {
	// Ignore cross package function call, since the function call in other
	// package will be handled in that package. The final result will be composed
	// from all the passes.
	if fn.Package() != nil && fn.Package() != pkg {
		return
	}

	// Record those functions have been traversed, to avoid cyclic call.
	if _, ok := t.seen.functions[fn]; ok {
		return
	}
	t.seen.functions[fn] = struct{}{}

	for _, param := range fn.Params {
		t.walkValue(pkg, param, cb)
	}

	t.walkInstructions(pkg, fn, cb)

	for _, anon := range fn.AnonFuncs {
		// functions use anonymous functions defined beneath them
		t.walkFunction(pkg, anon, cb)
	}
}

func (t *Traversal) walkInstructions(pkg *ssa.Package, fn *ssa.Function, cb WalkCallback) {
	for _, b := range fn.Blocks {
		for _, instr := range b.Instrs {
			if _, ok := t.seen.instructions[instr]; ok {
				continue
			}
			t.seen.instructions[instr] = struct{}{}

			// traverse the operands in instructions
			ops := instr.Operands(nil)

			for _, arg := range ops {
				t.walkValue(pkg, *arg, cb)
			}
		}
	}
}

func (t *Traversal) walkValue(pkg *ssa.Package, v ssa.Value, cb WalkCallback) {
	if v == nil {
		return
	}

	if _, ok := t.seen.values[v]; ok {
		return
	}
	t.seen.values[v] = struct{}{}

	phi, ok := v.(*ssa.Phi)
	if !ok {
		switch v := v.(type) {
		case *ssa.Function:
			t.walkFunction(pkg, v, cb)
			return
		}
		cb(v)
		return
	}

	applyPhi := func(v *ssa.Phi) {
		for _, e := range v.Edges {
			t.walkValue(pkg, e, cb)
		}
	}
	applyPhi(phi)
}
