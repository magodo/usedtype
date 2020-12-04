package usedtype

import (
	"go/types"
	"sort"
	"strings"
)

const honorJSONSkip = true

type StructFieldFullUsage struct {
	dm             StructDirectUsageMap
	field          structField
	nestedFields   map[structField]StructFieldFullUsage
	seenStructures map[*types.Named]struct{}
}

type StructFullUsages map[*types.Named]StructFullUsage

type StructFullUsage struct {
	root         *types.Named
	nestedFields map[structField]StructFieldFullUsage
}

func (fu StructFullUsage) String() string {
	if len(fu.nestedFields) == 0 {
		return ""
	}
	var out = []string{fu.root.String()}
	indexes := make([]int, len(fu.nestedFields))
	tmpM := map[int]StructFieldFullUsage{}
	cnt := 0
	for k, v := range fu.nestedFields {
		indexes[cnt] = k.index
		tmpM[k.index] = v
		cnt++
	}
	sort.Ints(indexes)

	for _, idx := range indexes {
		out = append(out, tmpM[idx].StringWithIndent(2))
	}
	return strings.Join(out, "\n")
}

func (fus StructFullUsages) String() string {
	var keys comparableNamed = make([]*types.Named, len(fus))
	cnt := 0
	for k := range fus {
		keys[cnt] = k
		cnt++
	}
	sort.Sort(keys)
	var out []string
	for _, key := range keys {
		if fus[key].String() == "" {
			continue
		}
		out = append(out, fus[key].String())
	}
	return strings.Join(out, "\n")
}

func (ffu StructFieldFullUsage) String() string {
	return ffu.StringWithIndent(0)
}

func (ffu StructFieldFullUsage) StringWithIndent(ident int) string {
	prefix := strings.Repeat("  ", ident)
	var out = []string{prefix + ffu.field.String()}

	indexes := make([]int, len(ffu.nestedFields))
	tmpM := map[int]StructFieldFullUsage{}
	cnt := 0
	for k, v := range ffu.nestedFields {
		indexes[cnt] = k.index
		tmpM[k.index] = v
		cnt++
	}
	sort.Ints(indexes)

	for _, idx := range indexes {
		out = append(out, tmpM[idx].StringWithIndent(ident+2))
	}
	return strings.Join(out, "\n")
}

func (ffu *StructFieldFullUsage) buildStructFieldFullUsage(field structField) {
	t := DereferenceRElem(field.base.Field(field.index).Type())
	nt, ok := t.(*types.Named)
	if !ok {
		return
	}

	if _, ok := ffu.seenStructures[nt]; ok {
		return
	}
	ffu.seenStructures[nt] = struct{}{}

	du, ok := ffu.dm[nt]
	if !ok {
		return
	}

	for nestedField := range du {
		newffu := StructFieldFullUsage{
			dm:             ffu.dm,
			field:          nestedField,
			nestedFields:   map[structField]StructFieldFullUsage{},
			seenStructures: map[*types.Named]struct{}{},
		}
		for k, v := range ffu.seenStructures {
			newffu.seenStructures[k] = v
		}

		if nestedField.IsUnderlyingNamedStructOrArrayOfNamedStruct() {
			newffu.buildStructFieldFullUsage(nestedField)
		}
		ffu.nestedFields[nestedField] = newffu
	}
}

func BuildStructFullUsage(dm StructDirectUsageMap, root *types.Named) StructFullUsage {
	u := StructFullUsage{
		root:         root,
		nestedFields: map[structField]StructFieldFullUsage{},
	}

	du, ok := dm[root]
	if !ok {
		return u
	}

	for nestedField := range du {
		ffu := StructFieldFullUsage{
			dm:             dm,
			field:          nestedField,
			nestedFields:   map[structField]StructFieldFullUsage{},
			seenStructures: map[*types.Named]struct{}{root: {}},
		}
		if nestedField.IsUnderlyingNamedStructOrArrayOfNamedStruct() {
			ffu.buildStructFieldFullUsage(nestedField)
		}
		u.nestedFields[nestedField] = ffu
	}
	return u
}

func BuildStructFullUsages(dm StructDirectUsageMap, rootSet StructSet) StructFullUsages {
	us := map[*types.Named]StructFullUsage{}
	for root := range rootSet {
		us[root] = BuildStructFullUsage(dm, root)
	}
	return us
}
