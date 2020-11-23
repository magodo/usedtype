package usedtype

import (
	"fmt"
	"go/types"
	"golang.org/x/tools/go/ssa"
	"reflect"
	"strings"
)

type structField struct {
	index int
	t     types.Type
}

type structFields []structField

func (fields structFields) String() string {
	var underlyingNamed func(t types.Type) *types.Named
	underlyingNamed = func(t types.Type) *types.Named {
		switch t := t.(type) {
		case *types.Named:
			return t
		case *types.Pointer:
			return underlyingNamed(t.Elem())
		default:
			panic("Not gonna happen")
		}
	}

	underlyingStruct := func(t types.Type) *types.Struct {
		named := underlyingNamed(t)
		switch v := named.Underlying().(type) {
		case *types.Struct:
			return v
		default:
			panic("Not gonna happen")
		}
	}

	fieldStrs := []string{}
	for idx, field := range fields {
		if idx == 0 {
			named := underlyingNamed(field.t)
			fieldStrs = append(fieldStrs, named.Obj().Name())
		}

		strct := underlyingStruct(field.t)
		tag := reflect.StructTag(strct.Tag(field.index))
		jsonTag := tag.Get("json")
		idx := strings.Index(jsonTag, ",")
		var fieldName string
		if idx == -1 {
			fieldName = jsonTag
		} else {
			fieldName = jsonTag[:idx]
			if fieldName == "" {
				fieldName = strct.Field(field.index).Name()
			}
		}
		fieldStrs = append(fieldStrs, fieldName)
	}
	return strings.Join(fieldStrs, ".")
}

type UseDefBranch struct {
	// root is the starting Value of this branch
	root ssa.Value

	// Instr represents the current instruction in the use-def chain
	instr ssa.Instruction

	// fields keep any struct field in the use-def chain
	fields structFields

	// seenInstructions keep all the instructions met till now, to avoid cyclic reference
	seenInstructions map[ssa.Instruction]struct{}

	// seenValues keep all the Values met till now, to avoid cyclic reference
	seenValues map[ssa.Value]struct{}

	// end means this use-def chain reaches the end
	end bool
}

func (branch UseDefBranch) String() string {
	instr := ""
	if branch.instr != nil {
		instr = branch.instr.String()
	}
	fields := branch.fields.String()
	return fmt.Sprintf("%s [%s]", instr, fields)
}

type UseDefBranches []UseDefBranch

func NewUseDefBranches(instr ssa.Instruction, value ssa.Value) UseDefBranches {
	tmpBranch := UseDefBranch{
		root:   value,
		fields: []structField{},
		// NOTE: here we regard the surrounding Instruction of the Value as seen. This avoids that we get this instruction
		// back when getting the source referrers of the value.
		seenInstructions: map[ssa.Instruction]struct{}{instr: {}},
		seenValues:       map[ssa.Value]struct{}{},
	}

	refinstrs := tmpBranch.sourceReferrersOfValue(value)
	if refinstrs == nil {
		return nil
	}
	var branches UseDefBranches
	for _, refinstr := range *refinstrs {
		// Since we do not have a starting Instruction (only a starting Value), after we get the referrer instructions,
		// we will need to check that the referrer instruction of the Value is not the Value itself (only if the Value is an
		// Instruction at the same time).
		// This is non-fatal, but makes the final output branches neat.
		if vintr, ok := value.(ssa.Instruction); ok {
			if vintr == refinstr {
				continue
			}
		}

		branches = append(branches, tmpBranch.propagate(refinstr))
	}
	return branches
}

// Walk walks the use-def chain in backward for each input branch. In each pass,
// each branch will move one step backward in the use-def chain, which might either
// return the branch itself back (means this branch has ended), or return several new
// branches which diverge because of the Value under used is got "defined" in multiple
// places (in this case, it is because the structure/sub-structure's members are defined
// in different places).
func (branches UseDefBranches) Walk() UseDefBranches {
	// Check whether we still have any branch to go on iterating.
	var toContinue bool
	for _, branch := range branches {
		if !branch.end {
			toContinue = true
		}
	}
	if !toContinue {
		return branches
	}

	// Iterate the branches.
	var newBranches UseDefBranches
	for _, branch := range branches {
		if branch.end {
			newBranches = append(newBranches, branch)
			continue
		}
		nextBranches := branch.next()
		for _, nextBranch := range nextBranches {
			newBranches = append(newBranches, nextBranch)
		}
	}

	return newBranches.Walk()
}

// next move one step backward in the use-def chain to the next def point and return the new set of use-def branches.
// If there is no new def point (referrer) back in the chain, the current branch is returned with the "end" set to true.
func (branch UseDefBranch) next() UseDefBranches {
	referInstrs := branch.sourceReferrersOfInstruction(branch.instr)

	// In case current instruction has no referrer, it means current use-def branch reaches to the end.
	// This is possible in cases like "Const" instruction.
	if referInstrs == nil {
		branch.end = true
		return []UseDefBranch{branch}
	}

	var nextBranches UseDefBranches

	for _, instr := range *referInstrs {
		nextBranches = append(nextBranches, branch.propagate(instr))
	}

	return nextBranches
}

// propagate propagate a new branch from an instruction and an existing branch.
// If that instruction is a FieldAddr, it will additionally add the field info to the new branch.
// If that instruction is seen before, then current branch will be returned back with "end" marked.
func (branch UseDefBranch) propagate(instr ssa.Instruction) UseDefBranch {
	if branch.end {
		panic(fmt.Sprintf("%s: ended branch can't propagate new branch", branch))
	}

	if _, ok := branch.seenInstructions[instr]; ok {
		branch.end = true
		return branch
	}

	newSeenInstructions := map[ssa.Instruction]struct{}{}
	for k, v := range branch.seenInstructions {
		newSeenInstructions[k] = v
	}

	newSeenValues := map[ssa.Value]struct{}{}
	for k, v := range branch.seenValues {
		newSeenValues[k] = v
	}

	newFields := make([]structField, len(branch.fields))
	copy(newFields, branch.fields)

	switch instr := instr.(type) {
	case *ssa.FieldAddr:
		newFields = append(newFields, structField{
			index: instr.Field,
			t:     instr.X.Type(),
		})
	}
	return UseDefBranch{
		root:             branch.root,
		instr:            instr,
		fields:           newFields,
		seenInstructions: newSeenInstructions,
		seenValues:       newSeenValues,
	}
}

// sourceReferrersOfInstruction return the instructions that defines the used value in the instruction.
func (branch *UseDefBranch) sourceReferrersOfInstruction(instr ssa.Instruction) *[]ssa.Instruction {
	if _, ok := branch.seenInstructions[instr]; ok {
		return nil
	}
	branch.seenInstructions[instr] = struct{}{}
	switch instr := instr.(type) {
	case *ssa.Extract:
		return branch.sourceReferrersOfValue(instr)
	case *ssa.UnOp:
		return branch.sourceReferrersOfValue(instr)
	case *ssa.FieldAddr:
		return branch.sourceReferrersOfValue(instr)
	case *ssa.Phi:
		return branch.sourceReferrersOfValue(instr)
	case *ssa.Call:
		return branch.sourceReferrersOfValue(instr)
	case *ssa.MakeMap:
		return branch.sourceReferrersOfValue(instr)
	case *ssa.Return:
		var instrs []ssa.Instruction
		for _, result := range instr.Results {
			srcinstrs := branch.sourceReferrersOfValue(result)
			if srcinstrs != nil {
				instrs = append(instrs, *srcinstrs...)
			}
		}
		return &instrs
	case *ssa.Store:
		return branch.sourceReferrersOfValue(instr.Val)
	default:
		panic("TODO: " + instr.String())
	}
}

// sourceReferrersOfValue return the instructions that defines the used value in the Value.
func (branch *UseDefBranch) sourceReferrersOfValue(value ssa.Value) *[]ssa.Instruction {
	if _, ok := branch.seenValues[value]; ok {
		return nil
	}
	branch.seenValues[value] = struct{}{}

	switch value := value.(type) {
	case *ssa.Alloc:
		return value.Referrers()
	case *ssa.BinOp:
		var referrers []ssa.Instruction
		xref := branch.sourceReferrersOfValue(value.X)
		if xref != nil {
			referrers = append(referrers, *xref...)
		}
		yref := branch.sourceReferrersOfValue(value.Y)
		if yref != nil {
			referrers = append(referrers, *yref...)
		}
		return &referrers
	case *ssa.Const:
		return value.Referrers()
	case *ssa.Extract:
		return value.Tuple.Referrers()
	case *ssa.UnOp:
		return branch.sourceReferrersOfValue(value.X)
	case *ssa.FieldAddr:
		return value.Referrers()
	case *ssa.Phi:
		var referrers []ssa.Instruction
		for _, edge := range value.Edges {
			srcreferrers := branch.sourceReferrersOfValue(edge)
			if srcreferrers != nil {
				referrers = append(referrers, *srcreferrers...)
			}
		}
		return &referrers
	case *ssa.Call:
		callcomm := value.Common()
		if callcomm.Method == nil {
			// call mode
			switch v := callcomm.Value.(type) {
			case *ssa.Function:
				var instrs []ssa.Instruction
				for _, b := range v.Blocks {
					// The return instruction is guaranteed to be the last instruction in each BasicBlock
					if instr, ok := b.Instrs[len(b.Instrs)-1].(*ssa.Return); ok {
						referrers := branch.sourceReferrersOfInstruction(instr) // TODO: Will there be cyclic ref?
						if referrers != nil {
							instrs = append(instrs, *referrers...)
						}
					}
				}
				return &instrs
			case *ssa.MakeClosure:
				panic("TODO:" + value.String())
			case *ssa.Builtin:
				panic("TODO:" + value.String())
			default:
				panic("should not reach here")
			}
		} else {
			// invoke mode (dynamic dispatch on interface)
			panic("TODO:" + value.String())
		}
	case *ssa.MakeMap:
		return nil
	default:
		panic("TODO:" + value.String())
	}
}
