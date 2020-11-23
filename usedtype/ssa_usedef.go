package usedtype

import (
	"fmt"
	"go/token"
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

		// This field is ignored in json request
		if fieldName == "-" {
			continue
		}
		fieldStrs = append(fieldStrs, fieldName)
	}
	return strings.Join(fieldStrs, ".")
}

type fromValue struct {
	value ssa.Value

	// refCount is used to keep track how many times the value is referenced (&).
	// This will be added each time it is referenced (&), and will be reduced each time
	// it is de-referenced (*).
	refCount int
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

	fromValue fromValue

	// for debug purpose only
	fset *token.FileSet
}

type UseDefBranches []UseDefBranch

func NewUseDefBranches(instr ssa.Instruction, value ssa.Value, fset *token.FileSet) UseDefBranches {
	tmpBranch := UseDefBranch{
		root:      value,
		fromValue: fromValue{value, 0},
		fields:    []structField{},
		// NOTE: here we regard the surrounding Instruction of the Value as seen. This avoids that we get this instruction
		// back when getting the source referrers of the value.
		seenInstructions: map[ssa.Instruction]struct{}{instr: {}},
		seenValues:       map[ssa.Value]struct{}{},
		fset:             fset,
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

	// DEBUG

	//var debugMsgs = []string{}
	//for _, b := range branches {
	//	debugMsgs = append(debugMsgs, b.String())
	//}
	//log.Printf("Walking on: \n%s", strings.Join(debugMsgs, "\n"))

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
		fromValue:        branch.fromValue,
		instr:            instr,
		fields:           newFields,
		seenInstructions: newSeenInstructions,
		seenValues:       newSeenValues,
		fset:             branch.fset,
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
	case *ssa.IndexAddr:
		return branch.sourceReferrersOfValue(instr)
	case *ssa.Phi:
		return branch.sourceReferrersOfValue(instr)
	case *ssa.Call:
		return branch.sourceReferrersOfValue(instr)
	case *ssa.MakeMap:
		return branch.sourceReferrersOfValue(instr)
	case *ssa.TypeAssert:
		return branch.sourceReferrersOfValue(instr)
	case *ssa.ChangeType:
		return branch.sourceReferrersOfValue(instr)
	case *ssa.Convert:
		return branch.sourceReferrersOfValue(instr)
	case *ssa.Slice:
		return branch.sourceReferrersOfValue(instr)
	case *ssa.MakeSlice:
		return branch.sourceReferrersOfValue(instr)
	case *ssa.MakeChan:
		return branch.sourceReferrersOfValue(instr)
	case *ssa.MakeInterface:
		return branch.sourceReferrersOfValue(instr)
	case *ssa.MakeClosure:
		return branch.sourceReferrersOfValue(instr)
	case *ssa.Lookup:
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
		branch.fromValue = fromValue{
			value:    instr.Val,
			refCount: branch.fromValue.refCount - 1,
		}
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
		branch.fromValue.value = value
		branch.fromValue.refCount++
		return value.Referrers()
	case *ssa.BinOp:
		branch.fromValue.value = value

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
	case *ssa.Extract:
		branch.fromValue.value = value
		return branch.sourceReferrersOfValue(value.Tuple)
	case *ssa.UnOp:
		switch value.Op {
		case token.NOT,
			token.SUB,
			token.ARROW,
			token.XOR:
			branch.fromValue.value = value
			return branch.sourceReferrersOfValue(value.X)
		case token.MUL:
			// In below expression:
			// Y = *X

			// In case the from value is the UnOp Value itself, we expect data flow: Y <- X.
			if branch.fromValue.value == value {
				branch.fromValue.value = value
				branch.fromValue.refCount--
				return branch.sourceReferrersOfValue(value.X)
			}

			// Otherwise, it means from value is X, then we expect data flow: Y->X.

			// refCount is > 0 means the X is the address of some Value, so Y can plays a role
			// as value source (def).
			if branch.fromValue.refCount > 0 {
				branch.fromValue.value = value
				branch.fromValue.refCount--
				return value.Referrers()
			}

			// refCount is 0 means the X is not an address, means the data flow ends here (Y will be used in other places though, that should
			// be handled in other branches / passes).
			return nil
		}
		return branch.sourceReferrersOfValue(value.X)
	case *ssa.FieldAddr:
		branch.fromValue.value = value
		branch.fromValue.refCount++
		return value.Referrers()
	case *ssa.IndexAddr:
		branch.fromValue.value = value
		branch.fromValue.refCount++
		return value.Referrers()
	case *ssa.Phi:
		var referrers []ssa.Instruction
		for _, edge := range value.Edges {
			branch.fromValue.value = edge
			srcreferrers := branch.sourceReferrersOfValue(edge)
			if srcreferrers != nil {
				referrers = append(referrers, *srcreferrers...)
			}
		}
		return &referrers
	case *ssa.Call:
		branch.fromValue.value = value
		callcomm := value.Common()
		if callcomm.IsInvoke() {
			// invoke mode (dynamic dispatch on interface)
			// TODO: figure out how to get the concrete Call instead of the interface abstract method
			return nil
		} else {
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
				// TODO: There is no way to get the operands of builtin-function easily, so we are returning nil here for now.
				return nil
			default:
				panic("should not reach here")
			}
		}
	case *ssa.Parameter:
		return nil
	case *ssa.TypeAssert:
		return nil
	case *ssa.ChangeType:
		return nil
	case *ssa.Convert:
		branch.fromValue.value = value
		return branch.sourceReferrersOfValue(value.X)
	case *ssa.Slice:
		branch.fromValue.value = value
		return branch.sourceReferrersOfValue(value.X)
	case *ssa.MakeMap:
		return nil
	case *ssa.MakeSlice:
		return nil
	case *ssa.MakeChan:
		return nil
	case *ssa.MakeInterface:
		return nil
	case *ssa.MakeClosure:
		return nil
	case *ssa.FreeVar:
		return nil
	case *ssa.Const:
		return nil
	case *ssa.Lookup:
		branch.fromValue.value = value
		return branch.sourceReferrersOfValue(value.X)
	default:
		panic("TODO:" + value.String())
	}
}

func (branch UseDefBranch) String() string {
	instr := ""
	pos := "-"
	if branch.instr != nil {
		instr = branch.instr.String()
		pos = branch.fset.Position(branch.instr.Pos()).String()
	}
	fields := branch.fields.String()
	return fmt.Sprintf("%s (%s) [%s]", instr, pos, fields)
}
