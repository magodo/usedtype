package usedtype

import (
	"fmt"
	"go/token"
	"go/types"
	"reflect"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"

	"golang.org/x/tools/go/ssa"
)

const honorJSONSkip = true

type structField struct {
	base  *types.Struct
	index int
}

func (u structField) Exported() bool {
	return u.base.Field(u.index).Exported()
}

func (u structField) IsUnderlyingNamedStructOrArrayOfNamedStruct() bool {
	t := u.base.Field(u.index).Type()
	return IsUnderlyingNamedStructOrArrayOfNamedStruct(t)
}

func (u structField) String() string {
	fieldName := u.base.Field(u.index).Name()
	tag := reflect.StructTag(u.base.Tag(u.index))
	jsonTag := tag.Get("json")
	idx := strings.Index(jsonTag, ",")
	var jsonTagName string
	if idx == -1 {
		jsonTagName = jsonTag
	} else {
		jsonTagName = jsonTag[:idx]
		if jsonTagName == "" {
			jsonTagName = u.base.Field(u.index).Name()
		}
	}

	return fmt.Sprintf("%s (%s)", fieldName, jsonTagName)
}

type structFieldSet map[structField][]token.Position

func (us structFieldSet) String() string {
	type structFieldUsage struct {
		field structField
		pos   []token.Position
	}

	indexes := []int{}
	tmpM := map[int]structFieldUsage{}
	for k, v := range us {
		indexes = append(indexes, k.index)
		tmpM[k.index] = structFieldUsage{
			field: k,
			pos:   v,
		}
	}

	output := []string{}
	for _, index := range indexes {
		field := tmpM[index].field.String()
		usages := []string{}
		for _, pos := range tmpM[index].pos {
			usages = append(usages, pos.String())
		}
		output = append(output, fmt.Sprintf("%s\n\t%s\n", field, strings.Join(usages, "\n\t")))
	}
	return strings.Join(output, "\n")
}

type StructDirectUsageMap map[*types.Named]structFieldSet

func (u StructDirectUsageMap) String() string {
	var keys comparableNamed = make([]*types.Named, len(u))
	cnt := 0
	for k := range u {
		keys[cnt] = k
		cnt++
	}
	sort.Sort(keys)
	var out = []string{}
	for _, k := range keys {
		out = append(out, fmt.Sprintf("%s\n===\n%s\n", k.String(), u[k].String()))
	}
	return strings.Join(out, "\n")
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

func FindInPackageStructureDirectUsage(pkgs []*packages.Package, ssapkgs []*ssa.Package) StructDirectUsageMap {
	output := StructDirectUsageMap{}
	for idx := range ssapkgs {
		ssapkg := ssapkgs[idx]
		pkg := pkgs[idx]
		var cb WalkInstrCallback
		cb = func(instr ssa.Instruction) {
			switch instr := instr.(type) {
			case *ssa.FieldAddr:
				output.record(pkg, instr, instr.X, instr.Field)
			case *ssa.Field:
				output.record(pkg, instr, instr.X, instr.Field)
			}
		}
		ssaTraversal := NewTraversal()
		ssaTraversal.WalkInPackage(ssapkg, cb, nil)
	}

	return output
}

func (m StructDirectUsageMap) record(pkg *packages.Package, instr ssa.Instruction, value ssa.Value, index int) {
	t := DereferenceRElem(value.Type())
	if !IsUnderlyingNamedStruct(t) {
		return
	}
	nt := t.(*types.Named)
	st := nt.Underlying().(*types.Struct)
	u := structField{
		base:  st,
		index: index,
	}
	if !u.Exported() {
		return
	}
	if len(m[nt]) == 0 {
		m[nt] = map[structField][]token.Position{}
	}
	pos := instr.Pos()
	if pos == token.NoPos {
		pos = value.Pos()
	}
	m[nt][u] = append(m[nt][u], pkg.Fset.Position(pos))
}

type StructFullUsage struct {
	root         *types.Named
	nestedFields map[structField]StructFieldFullUsage
}

type StructFieldFullUsage struct {
	dm             StructDirectUsageMap
	field          structField
	nestedFields   map[structField]StructFieldFullUsage
	seenStructures map[*types.Named]struct{}
}

func (fu StructFullUsage) String() string {
	var out = []string{fu.root.String()}
	for _, nestedField := range fu.nestedFields {
		out = append(out, nestedField.StringWithIndent(2))
	}
	return strings.Join(out, "\n")
}

func (ffu StructFieldFullUsage) String() string {
	return ffu.StringWithIndent(0)
}

func (ffu StructFieldFullUsage) StringWithIndent(ident int) string {
	prefix := strings.Repeat("  ", ident)
	var out = []string{prefix + ffu.field.String()}
	for _, nestedField := range ffu.nestedFields {
		out = append(out, nestedField.StringWithIndent(ident+2))
	}
	return strings.Join(out, "\n")
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
		if IsUnderlyingNamedStructOrArrayOfNamedStruct(nestedField.base.Field(nestedField.index).Type()) {
			ffu.buildStructFieldFullUsage(nestedField)
		}
		u.nestedFields[nestedField] = ffu
	}
	return u
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

		if IsUnderlyingNamedStructOrArrayOfNamedStruct(nestedField.base.Field(nestedField.index).Type()) {
			newffu.buildStructFieldFullUsage(nestedField)
		}
		ffu.nestedFields[nestedField] = newffu
	}
}
