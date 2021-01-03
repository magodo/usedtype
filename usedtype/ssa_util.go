package usedtype

import "golang.org/x/tools/go/ssa"

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
