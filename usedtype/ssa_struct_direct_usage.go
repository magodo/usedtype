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

type structField struct {
	base  *types.Struct // This is always an underlying type of a Named type, so it is canonical.:w
	index int
}

func (u structField) Exported() bool {
	return u.base.Field(u.index).Exported()
}

func (u structField) DereferenceRElem() types.Type {
	return DereferenceRElem(u.base.Field(u.index).Type())
}

func (u structField) IsElemUnderlyingNamedStructOrInterface() bool {
	t := u.base.Field(u.index).Type()
	return IsElemUnderlyingNamedStructOrInterface(t)
}

func (u structField) IsElemUnderlyingNamedInterface() bool {
	t := u.base.Field(u.index).Type()
	return IsElemUnderlyingNamedInterface(t)
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

type StructDirectUsageMap map[*types.Named]structDirectUsage

func (u StructDirectUsageMap) String() string {
	var keys namedTypes = make([]*types.Named, len(u))
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

type structDirectUsage map[structField][]token.Position

func (us structDirectUsage) String() string {
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
