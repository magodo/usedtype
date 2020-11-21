package ssa

import (
	"go/types"
	"golang.org/x/tools/go/ssa"
)

type structField struct {
	index int
	t     types.Type
}

type UseDefBranch struct {
	// instr represents the current instruction in the use-def chain
	instr ssa.Instruction

	// fields keep any struct field in the use-def chain
	fields []structField

	// seenInstructions keep all the instructions met till now, to avoid cyclic reference
	seenInstructions map[ssa.Instruction]struct{}

	// end means this use-def chain reaches the end
	end bool
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

// previous move one step backward in the use-def chain to the previous def point and return the new set of use-def branches.
// If there is no new def point (referrer) back in the chain, the current branch is returned with the "end" set to true.
func (branch UseDefBranch) previous() UseDefBranches {
	referInstrs := sourceReferrers(branch.instr)

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
	for _, ctx := range branches {
		if !ctx.end {
			toContinue = true
		}
	}

	if !toContinue {
		return branches
	}

	var newBranches UseDefBranches
	for _, branch := range branches {
		prevBranches := branch.previous()
		for _, prevBranch := range prevBranches {
			newBranches = append(newBranches, prevBranch)
		}
	}

	return newBranches.Walk()
}

// sourceReferrers return the instructions that defines the used value in the instruction.
func sourceReferrers(instr ssa.Instruction) *[]ssa.Instruction {
	switch instr := instr.(type) {
	case *ssa.UnOp:
		return instr.X.Referrers()
	case *ssa.FieldAddr:
		return instr.Referrers()
	case *ssa.Store:
		return instr.Val.Referrers()
	default:
		// TODO
		ret := []ssa.Instruction{}
		return &ret
	}
}
