package ssa

import (
	"fmt"
	"golang.org/x/tools/go/ssa"
)

type Seen struct {
	functions map[*ssa.Function]struct{}
}

var seen = Seen{
	functions: map[*ssa.Function]struct{}{},
}

type WalkCallback func(instr ssa.Instruction, val ssa.Value)

// WalkInPackage traverse inside a ssa package from all top level functions (skipping other top level members:
// Type, NamedConst and Global). It will iterate each instruction and the value belongs to it.
// Note that only the functions defined in this ssa package is traversed, it will not cross package boundary.
func WalkInPackage(pkg *ssa.Package, cb WalkCallback) {
	for _, m := range pkg.Members {
		switch m := m.(type) {
		case *ssa.Type,
			*ssa.NamedConst:
			// nothing to do, since it will not appear any Value of target type
		case *ssa.Global:
			// TODO
		case *ssa.Function:
			walkFunction(pkg, m, cb)
		default:
			panic(fmt.Sprintf("unreachable: %T", m))
		}
	}
}

func walkFunction(pkg *ssa.Package, fn *ssa.Function, cb WalkCallback) {
	// Ignore cross package function call, since the function call in other
	// package will be handled in that package. The final result will be composed
	// from all the passes.
	if fn.Package() != nil && fn.Package() != pkg {
		return
	}

	// Record those functions have been traversed, to avoid cyclic call.
	if _, ok := seen.functions[fn]; ok {
		return
	}
	seen.functions[fn] = struct{}{}

	walkInstructions(pkg, fn, cb)

	for _, anon := range fn.AnonFuncs {
		// functions use anonymous functions defined beneath them
		walkFunction(pkg, anon, cb)
	}
}

func walkInstructions(pkg *ssa.Package, fn *ssa.Function, cb WalkCallback) {
	for _, b := range fn.Blocks {
		for _, instr := range b.Instrs {

			// traverse the operands in instructions

			ops := instr.Operands(nil)

			switch instr.(type) {
			case *ssa.Store:
				// ignore the first operands for Store, as that represents the Addr
				ops = ops[1:]
			case *ssa.DebugRef:
				// ignore ops for debug ref
				ops = nil
			}

			for _, arg := range ops {
				walkPhi(pkg, instr, *arg, cb)
			}
		}
	}
}

func walkPhi(pkg *ssa.Package, instr ssa.Instruction, v ssa.Value, cb func(instr ssa.Instruction, v ssa.Value)) {
	phi, ok := v.(*ssa.Phi)
	if !ok {
		switch v := v.(type) {
		case *ssa.Function:
			walkFunction(pkg, v, cb)
		}
		cb(instr, v)
		return
	}

	seen := map[ssa.Value]struct{}{}
	var applyPhi func(v *ssa.Phi)
	applyPhi = func(v *ssa.Phi) {
		if _, ok := seen[v]; ok {
			return
		}
		seen[v] = struct{}{}
		for _, e := range v.Edges {
			walkPhi(pkg, instr, e, cb)

			//switch e := e.(type) {
			//case *ssa.Phi:
			//	applyPhi(e)
			//case *ssa.Function:
			//	walkFunction(pkg, e, cb)
			//default:
			//	cb(instr, e)
			//}
		}
	}
	applyPhi(phi)
}
