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
	// Instr represents the current instruction in the use-def chain
	instr ssa.Instruction

	// fields keep any struct field in the use-def chain
	fields structFields

	// seenInstructions keep all the instructions met till now, to avoid cyclic reference
	seenInstructions map[ssa.Instruction]struct{}

	// end means this use-def chain reaches the end
	end bool
}

func (b UseDefBranch) String() string {
	instr := b.instr.String()
	fields := b.fields.String()
	return fmt.Sprintf("%s [%s]", instr, fields)
}

type UseDefBranches []UseDefBranch

func NewUseDefBranch(instr ssa.Instruction, value ssa.Value) UseDefBranch {
	vinstr, ok := value.(ssa.Instruction)
	if !ok {
		panic("The starting node is not an Instruction")
	}
	return UseDefBranch{
		instr:            vinstr,
		fields:           []structField{},
		seenInstructions: map[ssa.Instruction]struct{}{instr: {}, vinstr: {}},
	}
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

	// This is to avoid duplicate referrer instructions occur during iteration.
	// It may contain duplicates if an instruction has a repeated operand.
	seenInstructions := map[ssa.Instruction]struct{}{}
	for k, v := range branch.seenInstructions {
		seenInstructions[k] = v
	}

	var nextBranches UseDefBranches

	for _, instr := range *referInstrs {
		if _, ok := seenInstructions[instr]; ok {
			continue
		}
		seenInstructions[instr] = struct{}{}

		newSeenInstructions := map[ssa.Instruction]struct{}{instr: {}}
		for k, v := range branch.seenInstructions {
			newSeenInstructions[k] = v
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
		nextBranches = append(nextBranches, UseDefBranch{
			instr:            instr,
			fields:           newFields,
			seenInstructions: newSeenInstructions,
		})
	}

	if len(nextBranches) == 0 {
		branch.end = true
		return []UseDefBranch{branch}
	}
	return nextBranches
}

// Walk walks the use-def chain in backward for each input branch. In each pass,
// each branch will move one step backward in the use-def chain, which might either
// return the branch itself back (means this branch has ended), or return several new
// branches which diverge because of the Value under used is got "defined" in multiple
// places (in this case, it is because the structure/sub-structure's members are defined
// in different places).
func (branches UseDefBranches) Walk() UseDefBranches {
	var toContinue bool
	for _, branch := range branches {
		if !branch.end {
			toContinue = true
		}
	}

	if !toContinue {
		return branches
	}

	var newBranches UseDefBranches
	for _, branch := range branches {
		nextBranches := branch.next()
		for _, nextBranch := range nextBranches {
			newBranches = append(newBranches, nextBranch)
		}
	}

	return newBranches.Walk()
}

// sourceReferrersOfInstruction return the instructions that defines the used value in the instruction.
func (branch *UseDefBranch) sourceReferrersOfInstruction(instr ssa.Instruction) *[]ssa.Instruction {
	// Record the instruction in the seen instruction set.
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
	// In case the value is an instruction itself, we will record it in the seen instruction set.
	if instr, ok := value.(ssa.Instruction); ok {
		branch.seenInstructions[instr] = struct{}{}
	}
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
