package usedtype

import (
	"fmt"
	"go/token"
	"go/types"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
)

type VirtAccessPoint struct {
	Pos   token.Position
	Instr ssa.Instruction
}

type StructDirectUsage map[StructField][]VirtAccessPoint

type StructDirectUsageMap map[*types.Named]StructDirectUsage

func (us StructDirectUsage) String() string {
	return us.StringWithIndent(0)
}

func (us StructDirectUsage) StringWithIndent(indent int) string {
	fieldPrefix := strings.Repeat(" ", indent)
	usagePrefix := strings.Repeat(" ", indent+2)

	type structFieldUsage struct {
		field StructField
		vps   []VirtAccessPoint
	}

	var indexes []int
	tmpM := map[int]structFieldUsage{}
	for k, v := range us {
		indexes = append(indexes, k.index)
		tmpM[k.index] = structFieldUsage{
			field: k,
			vps:   v,
		}
	}

	var output []string
	for _, index := range indexes {
		field := fieldPrefix + tmpM[index].field.String()
		var usages []string
		for _, vp := range tmpM[index].vps {
			usages = append(usages, usagePrefix+vp.Pos.String())
		}
		output = append(output, fmt.Sprintf("%s\n%s", field, strings.Join(usages, "\n")))
	}
	return strings.Join(output, "\n")
}

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
		out = append(out, fmt.Sprintf("%s\n%s", k.String(), u[k].StringWithIndent(2)))
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
	u := StructField{
		base:  st,
		index: index,
	}
	// ignore private field
	if !u.Exported() {
		return
	}
	if len(m[nt]) == 0 {
		m[nt] = map[StructField][]VirtAccessPoint{}
	}
	m[nt][u] = append(m[nt][u], VirtAccessPoint{
		Instr: instr,
		Pos:   InstrPosition(pkg.Fset, instr),
	})
}

// FindInPackageStructureDirectUsage searches among the ssapkgs to gather each virtual field access on exported fields
// for each Named struct.
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
