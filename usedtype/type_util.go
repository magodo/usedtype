package usedtype

import (
	"go/types"
)

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

func IsUnderlyingNamedStruct(t types.Type) bool {
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

func IsUnderlyingNamedStructOrArrayOfNamedStruct(t types.Type) bool {
	t = DereferenceR(t)
	switch t := t.(type) {
	case *types.Named:
		return IsUnderlyingNamedStruct(t)
	case *types.Array:
		return IsUnderlyingNamedStructOrArrayOfNamedStruct(t.Elem())
	case *types.Slice:
		return IsUnderlyingNamedStructOrArrayOfNamedStruct(t.Elem())
	default:
		return false
	}
}

type comparableNamed []*types.Named

func (st comparableNamed) Swap(i, j int) {
	st[i], st[j] = st[j], st[i]
}
func (st comparableNamed) Len() int {
	return len(st)
}
func (st comparableNamed) Less(i, j int) bool {
	return st[i].String() < st[j].String()
}

