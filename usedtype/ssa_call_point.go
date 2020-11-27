package usedtype

import "golang.org/x/tools/go/ssa"

type CallPointLookup struct {
	cache map[*ssa.Package]map[*ssa.Function][]ssa.Instruction
}

var callPointLookup = &CallPointLookup{
	cache: map[*ssa.Package]map[*ssa.Function][]ssa.Instruction{},
}

func (l *CallPointLookup)FindCallPoint(pkg *ssa.Package) map[*ssa.Function][]ssa.Instruction {
	if v, ok := l.cache[pkg]; ok {
		return v
	}
	c := map[*ssa.Function][]ssa.Instruction{}
	traverse := NewTraversal()
	traverse.WalkInPackage(pkg, func(instr ssa.Instruction, val ssa.Value) {
		switch instr := instr.(type) {
		case *ssa.Call:
			com := instr.Call
			if com.IsInvoke() {
				panic("TODO")
			}
			switch v := com.Value.(type) {
			case *ssa.Function:
				c[v] = append(c[v], instr)
				return
			case *ssa.MakeClosure:
				panic("TODO")
			case *ssa.Builtin:
				return
			default:
				panic("will never happen")
			}
		}
	})
	l.cache[pkg] = c
	return c
}
