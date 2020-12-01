package usedtype

import "golang.org/x/tools/go/ssa"

type CallPointLookup struct {
	cache map[*ssa.Package]map[*ssa.Function][]ssa.Instruction
}

var callPointLookup = &CallPointLookup{
	cache: map[*ssa.Package]map[*ssa.Function][]ssa.Instruction{},
}

func (l *CallPointLookup) FindCallPoint(pkg *ssa.Package) map[*ssa.Function][]ssa.Instruction {
	if v, ok := l.cache[pkg]; ok {
		return v
	}
	c := map[*ssa.Function][]ssa.Instruction{}
	traverse := NewTraversal()
	traverse.WalkInPackage(pkg, func(instr ssa.Instruction) {
		switch instr := instr.(type) {
		case *ssa.Call:
			com := instr.Call
			if com.IsInvoke() {
				if strict {
					panic("TODO")
				} else {
					return
				}
			}
			switch v := com.Value.(type) {
			case *ssa.Function:
				c[v] = append(c[v], instr)
				return
			case *ssa.MakeClosure:
				if strict {
					panic("TODO")
				} else {
					return
				}
			case *ssa.Builtin:
				return
			default:
				if strict {
					panic("TODO")
				} else {
					return
				}
			}
		}
	}, nil,
	)
	l.cache[pkg] = c
	return c
}
