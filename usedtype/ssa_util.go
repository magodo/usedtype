package usedtype

import (
	"go/token"

	"golang.org/x/tools/go/ssa"
)

func BBCanReach(start, end *ssa.BasicBlock) bool {
	seen := make(map[*ssa.BasicBlock]bool)
	var search func(b *ssa.BasicBlock) bool
	search = func(b *ssa.BasicBlock) bool {
		if !seen[b] {
			seen[b] = true
			if b == end {
				return true
			}
			for _, e := range b.Succs {
				if found := search(e); found != false {
					return found
				}
			}
		}
		return false
	}
	return search(start)
}

func InstrPosition(fset *token.FileSet, instr ssa.Instruction) token.Position {
	pos := instr.Pos()
	if pos != token.NoPos {
		return fset.Position(pos)
	}

	switch instr := instr.(type) {
	case *ssa.FieldAddr:
		// In case of composite literal, the the user facing position should be the one that assigns the field.
		referrers := instr.Referrers()
		if referrers != nil {
			for _, ref := range *referrers {
				store, ok := ref.(*ssa.Store)
				if !ok {
					continue
				}
				if store.Addr != instr {
					continue
				}
				return fset.Position(store.Pos())
			}
		}
		// fallback to the field owner's position, which is always available
		return fset.Position(instr.X.Pos())
	case *ssa.MakeInterface:
		return fset.Position(instr.X.Pos())
	case *ssa.Field:
		// In case of composite literal, the the user facing position should be the one that assigns the field.
		referrers := instr.Referrers()
		if referrers != nil {
			for _, ref := range *referrers {
				store, ok := ref.(*ssa.Store)
				if !ok {
					continue
				}
				if store.Addr != instr {
					continue
				}
				return fset.Position(store.Pos())
			}
		}
		// fallback to the field owner's position, which is always available
		return fset.Position(instr.X.Pos())
	default:
		panic("We should extend if panic")
	}
}
