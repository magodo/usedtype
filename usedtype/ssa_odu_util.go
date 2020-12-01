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
		// continue
	case *ssa.Global:
		// E.g. init$guard
		if v.Object() == nil {
			return false
		}
		// continue
	default:
		return false
	}

	return UnderlyingNamedStructOrArrayOfNamedStruct(v.Type())
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

// DereferenceR returns a pointer's element type; otherwise it returns
// T. If the element type is itself a pointer, DereferenceR will be
// applied recursively.
func DereferenceR(T types.Type) types.Type {
	if p, ok := T.Underlying().(*types.Pointer); ok {
		return DereferenceR(p.Elem())
	}
	return T
}

// DereferenceRElem is like DereferenceR, but it will continue to dereference the
// element if hit an array.
func DereferenceRElem(t types.Type) types.Type {
	t = DereferenceR(t)
	if arr, ok := t.(*types.Array); ok {
		return DereferenceRElem(arr.Elem())
	}
	if slice, ok := t.(*types.Slice); ok {
		return DereferenceRElem(slice.Elem())
	}
	return t
}

func UnderlyingNamedStruct(t types.Type) bool {
	t = DereferenceR(t)
	nt, ok := t.(*types.Named)
	if !ok {
		return false
	}
	if _, ok := nt.Underlying().(*types.Struct); !ok {
		return false
	}
	return true
}

func UnderlyingNamedStructOrArrayOfNamedStruct(t types.Type) bool {
	t = DereferenceR(t)
	switch t := t.(type) {
	case *types.Named:
		return UnderlyingNamedStruct(t)
	case *types.Array:
		return UnderlyingNamedStructOrArrayOfNamedStruct(t.Elem())
	case *types.Slice:
		return UnderlyingNamedStructOrArrayOfNamedStruct(t.Elem())
	default:
		return false
	}
}

func assert(b bool, msg ...string) {
	s := "will not happen"
	if !b {
		if len(msg) != 0 {
			s = fmt.Sprintf("%s: %s", s, strings.Join(msg, ","))
		}
		panic(s)
	}
}
