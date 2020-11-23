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

type DefUseBranch struct {
	// root is the starting Value of this branch
	root ssa.Value

	// refCount is used to keep track how many times current Value is referenced (&).
	// This will be added each time it is referenced (&), and will be reduced each time
	// it is de-referenced (*).
	refCount int

	// Instr represents the current instruction in the def-use chain
	instr ssa.Instruction

	// fields keep any struct field in the def-use chain
	fields structFields

	// seenInstructions keep all the instructions met till now, to avoid cyclic reference
	seenInstructions map[ssa.Instruction]struct{}

	// seenValues keep all the Values met till now, to avoid cyclic reference
	seenValues map[ssa.Value]struct{}

	// end means this def-use chain reaches the end
	end bool

	// for debug purpose only
	fset *token.FileSet
}

type DefUseBranches []DefUseBranch

func NewDefUseBranches(value ssa.Value, fset *token.FileSet) DefUseBranches {

	switch value.(type) {
	case *ssa.Alloc,
		*ssa.Global:
		// continue
	default:
		panic("value used to new DefUseBranch must be \"def\" node, which can be either *ssa.Alloc or *ssa.Global")
	}

	tmpBranch := DefUseBranch{
		root:             value,
		refCount:         0,
		fields:           []structField{},
		seenInstructions: map[ssa.Instruction]struct{}{},
		seenValues:       map[ssa.Value]struct{}{},
		fset:             fset,
	}

	refinstrs := tmpBranch.sourceReferrersOfValue(value)
	if refinstrs == nil {
		return nil
	}
	var branches DefUseBranches
	for _, refinstr := range *refinstrs {
		branches = append(branches, tmpBranch.propagate(refinstr))
	}
	return branches
}

// Walk walks the def-use chain in backward for each input branch. In each pass,
// each branch will move one step backward in the def-use chain, which might either
// return the branch itself back (means this branch has ended), or return several new
// branches which diverge because of the Value under used is got "defined" in multiple
// places (in this case, it is because the structure/sub-structure's members are defined
// in different places).
func (branches DefUseBranches) Walk() DefUseBranches {

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
	var newBranches DefUseBranches
	for _, branch := range branches {
		if branch.end {
			newBranches = append(newBranches, branch)
			continue
		}
		// get the next branch set by follow the def-use chain
		nextBranches := branch.next()
		for _, nextBranch := range nextBranches {
			newBranches = append(newBranches, nextBranch)
		}
	}

	return newBranches.Walk()
}

// next move one step forward in the def-use chain to the next def point and return the new set of def-use branches.
// If there is no new def point (referrer) in the chain, the current branch is returned with the "end" set to true.
func (branch DefUseBranch) next() DefUseBranches {
	referInstrs := branch.sourceReferrersOfInstruction(branch.instr)

	// In case current instruction has no referrer, it means current def-use branch reaches to the end.
	// This is possible in cases like "Const" instruction.
	if referInstrs == nil {
		branch.end = true
		return []DefUseBranch{branch}
	}

	var nextBranches DefUseBranches

	for _, instr := range *referInstrs {
		nextBranches = append(nextBranches, branch.propagate(instr))
	}

	return nextBranches
}

// propagate propagate a new branch from an instruction and an existing branch.
// If that instruction is a FieldAddr, it will additionally add the field info to the new branch.
// If that instruction is seen before, then current branch will be returned back with "end" marked.
func (branch DefUseBranch) propagate(instr ssa.Instruction) DefUseBranch {
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

	return DefUseBranch{
		root:             branch.root,
		refCount:         branch.refCount,
		instr:            instr,
		fields:           newFields,
		seenInstructions: newSeenInstructions,
		seenValues:       newSeenValues,
		fset:             branch.fset,
	}
}

// sourceReferrersOfInstruction return the instructions that defines the used value in the instruction.
func (branch *DefUseBranch) sourceReferrersOfInstruction(instr ssa.Instruction) *[]ssa.Instruction {
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
		// In a def-use chain, when the referrer is Store instruction, the used value is always the Addr.
		// TODO: need proof
		branch.refCount--
		return branch.sourceReferrersOfValue(instr.Val)
	default:
		panic("TODO: " + instr.String())
	}
}

// sourceReferrersOfValue return the instructions that defines the used value in the Value.
func (branch *DefUseBranch) sourceReferrersOfValue(value ssa.Value) *[]ssa.Instruction {
	if _, ok := branch.seenValues[value]; ok {
		return nil
	}
	branch.seenValues[value] = struct{}{}

	switch value := value.(type) {
	case *ssa.Alloc:
		vt := value.Type()
		cnt := 0
		for pt, ok := vt.(*types.Pointer); ok; {
			cnt++
			vt = pt.Elem()
			pt, ok = vt.(*types.Pointer)
		}
		branch.refCount += cnt
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
	case *ssa.Extract:
		return branch.sourceReferrersOfValue(value.Tuple)
	case *ssa.UnOp:
		switch value.Op {
		case token.NOT,
			token.SUB,
			token.ARROW,
			token.XOR:
			return branch.sourceReferrersOfValue(value.X)
		case token.MUL:
			branch.refCount--
			// refCount reaches 0 means this UnOp pass the Value to the left operand, which no longer continue the def-use chain from
			// the original owner.
			if branch.refCount == 0 {
				return nil
			}
			return value.Referrers()
		default:
			panic("won't reach here")
		}
	case *ssa.FieldAddr:
		vt := value.X.Type().(*types.Pointer).Elem().Underlying().(*types.Struct).Field(value.Field).Type()
		cnt := 1
		for pt, ok := vt.(*types.Pointer); ok; {
			cnt++
			vt = pt.Elem()
			pt, ok = vt.(*types.Pointer)
		}

		branch.refCount = cnt
		return value.Referrers()
	case *ssa.IndexAddr:
		// TODO: do we have to remember the fromValue to check whether the X or Index is used?
		branch.refCount++
		return value.X.Referrers()
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
					// TODO: if the return value ultimately sourced from parameter, we will need to go on analyzing parameters.
				}
				return &instrs
			case *ssa.MakeClosure:
				panic("TODO:" + value.String())
			case *ssa.Builtin:
				panic("TODO:" + value.String())
			default:
				panic("should not reach here")
			}
		}
	case *ssa.Convert:
		return branch.sourceReferrersOfValue(value.X)
	case *ssa.Slice:
		return branch.sourceReferrersOfValue(value.X)
	case *ssa.MakeInterface:
		return branch.sourceReferrersOfValue(value.X)
	case *ssa.Lookup:
		return branch.sourceReferrersOfValue(value.X)
	case *ssa.Global:
		vt := value.Type()
		cnt := 0
		for pt, ok := vt.(*types.Pointer); ok; {
			cnt++
			vt = pt.Elem()
			pt, ok = vt.(*types.Pointer)
		}
		branch.refCount += cnt
		return value.Referrers()
	case *ssa.Parameter:
		return nil
	case *ssa.TypeAssert:
		return nil
	case *ssa.ChangeType:
		return nil
	case *ssa.MakeMap:
		return nil
	case *ssa.MakeSlice:
		return nil
	case *ssa.MakeChan:
		return nil
	case *ssa.MakeClosure:
		return nil
	case *ssa.FreeVar:
		return nil
	case *ssa.Const:
		return nil
	default:
		panic("TODO:" + value.String())
	}
}

func (branch DefUseBranch) String() string {
	instr := ""
	pos := "-"
	if branch.instr != nil {
		instr = branch.instr.String()
		pos = branch.fset.Position(branch.instr.Pos()).String()
	}
	fields := branch.fields.String()
	return fmt.Sprintf("%s (%s) [%s]", instr, pos, fields)
}
