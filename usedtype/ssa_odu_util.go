package usedtype

import (
	"fmt"
	"go/types"
	"golang.org/x/tools/go/ssa"
	"strings"
)

// IsDefValue tells whether the input value is a "define" value.
func IsDefValue(v ssa.Value) bool {
	switch v := v.(type) {
	case *ssa.Alloc,
		*ssa.Parameter:
		return true
	case *ssa.Global:
		// E.g. init$guard
		if v.Object() == nil {
			return false
		}
		return true
	default:
		return false
	}
}

// ReferenceDepth return the reference level of a given SSA value.
func ReferenceDepth(vt types.Type) int {
	cnt := 0
	for pt, ok := vt.(*types.Pointer); ok; {
		cnt++
		vt = pt.Elem()
		pt, ok = vt.(*types.Pointer)
	}
	return cnt
}

func assert(b bool, msg... string) {
	s := "will not happen"
	if !b {
		if len(msg) != 0 {
			s = fmt.Sprintf("%s: %s", s, strings.Join(msg, ","))
		}
		panic(s)
	}
}
