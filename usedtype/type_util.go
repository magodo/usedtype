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

func IsUnderlyingNamedInterface(t types.Type) bool {
	t = DereferenceR(t)
	nt, ok := t.(*types.Named)
	if !ok {
		return false
	}
	if _, ok := nt.Underlying().(*types.Interface); !ok {
		return false
	}
	return true
}

func IsUnderlyingNamedStructOrInterface(t types.Type) bool {
	return IsUnderlyingNamedInterface(t) || IsUnderlyingNamedStruct(t)
}

func IsElemUnderlyingNamedInterface(t types.Type) bool {
	t = DereferenceR(t)
	switch t := t.(type) {
	case *types.Array:
		return IsElemUnderlyingNamedInterface(t.Elem())
	case *types.Slice:
		return IsElemUnderlyingNamedInterface(t.Elem())
	default:
		return IsUnderlyingNamedInterface(t)
	}
}

func IsElemUnderlyingNamedStructOrInterface(t types.Type) bool {
	t = DereferenceR(t)
	switch t := t.(type) {
	case *types.Array:
		return IsElemUnderlyingNamedStructOrInterface(t.Elem())
	case *types.Slice:
		return IsElemUnderlyingNamedStructOrInterface(t.Elem())
	default:
		return IsUnderlyingNamedStructOrInterface(t)
	}
}

type namedTypes []*types.Named

func (st namedTypes) Swap(i, j int) {
	st[i], st[j] = st[j], st[i]
}
func (st namedTypes) Len() int {
	return len(st)
}
func (st namedTypes) Less(i, j int) bool {
	return st[i].String() < st[j].String()
}

func AzureSDKTrack1Implements(v types.Type, nt *types.Named) bool {
	t, ok := nt.Underlying().(*types.Interface)
	if !ok {
		return false
	}
	// Store the struct types that implement this interface.
	implementors := []*types.Named{}

	// Store the interfaces that inherit this interface, including itself.
	interfaces := map[string]bool{
		nt.Obj().Name(): true,
	}

	for i := 0; i < t.NumMethods(); i++ {
		signature, ok := t.Method(i).Type().(*types.Signature)
		if !ok {
			continue
		}

		methodReturns := signature.Results()

		// The return type is always (struct ptr/interface, bool)
		if methodReturns.Len() != 2 {
			continue
		}

		vt := methodReturns.At(0).Type()
		vt = DereferenceR(vt)
		nt, ok := vt.(*types.Named)
		if !ok {
			continue
		}

		ut := nt.Underlying()

		switch ut.(type) {
		case *types.Interface:
			interfaces[nt.Obj().Name()] = true
		case *types.Struct:
			implementors = append(implementors, nt)
		}
	}

	for _, nt := range implementors {
		// Skip the hypothetic base types from the implementers
		if interfaces["Basic"+nt.Obj().Name()] {
			continue
		}
		if types.Identical(nt, v) {
			return true
		}
	}

	return false
}
