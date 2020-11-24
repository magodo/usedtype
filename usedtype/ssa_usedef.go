package usedtype

import (
	"fmt"
	"go/token"
	"go/types"
	"reflect"
	"strings"

	"golang.org/x/tools/go/ssa"
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
	// refCount is used to keep track how many times current Value is referenced (&).
	// This will be added each time it is referenced (&), and will be reduced each time
	// it is de-referenced (*).
	refCount int

	// valueChain keep track of the def-use value chain starting from the def value.
	valueChain []ssa.Value

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
		refCount:         0,
		valueChain:       []ssa.Value{},
		fields:           []structField{},
		seenInstructions: map[ssa.Instruction]struct{}{},
		seenValues:       map[ssa.Value]struct{}{},
		fset:             fset,
	}

	return tmpBranch.propagateOnValue(value)
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
		nextBranches := branch.propagateOnInstr(branch.instr)
		newBranches = append(newBranches, nextBranches...)
	}

	return newBranches.Walk()
}

func (branch DefUseBranch) copy() DefUseBranch {
	newValueChain := make([]ssa.Value, len(branch.valueChain))
	copy(newValueChain, branch.valueChain)

	newFields := make([]structField, len(branch.fields))
	copy(newFields, branch.fields)

	newSeenInstructions := map[ssa.Instruction]struct{}{}
	for k, v := range branch.seenInstructions {
		newSeenInstructions[k] = v
	}

	newSeenValues := map[ssa.Value]struct{}{}
	for k, v := range branch.seenValues {
		newSeenValues[k] = v
	}

	return DefUseBranch{
		valueChain:       newValueChain,
		refCount:         branch.refCount,
		fields:           newFields,
		seenInstructions: newSeenInstructions,
		seenValues:       newSeenValues,
		fset:             branch.fset,
	}
}

func (branch DefUseBranch) propagateOnReferrers(referrers *[]ssa.Instruction) DefUseBranches {
	var newBranches []DefUseBranch
	if referrers == nil {
		branch.end = true
		newBranches = []DefUseBranch{branch}
		return newBranches
	}

	for _, instr := range *referrers {
		b := branch.copy()
		b.instr = instr

		switch instr := instr.(type) {
		case *ssa.FieldAddr:
			b.fields = append(b.fields,
				structField{
					index: instr.Field,
					t:     instr.X.Type(),
				})
		}

		newBranches = append(newBranches, b)
	}
	return newBranches
}

// propagateOnInstr return the instructions that defines the used value in the instruction.
func (branch DefUseBranch) propagateOnInstr(instr ssa.Instruction) DefUseBranches {
	if _, ok := branch.seenInstructions[instr]; ok {
		branch.end = true
		return []DefUseBranch{branch}
	}
	branch.seenInstructions[instr] = struct{}{}
	switch instr := instr.(type) {
	case *ssa.Extract:
		return branch.propagateOnValue(instr)
	case *ssa.UnOp:
		return branch.propagateOnValue(instr)
	case *ssa.FieldAddr:
		return branch.propagateOnValue(instr)
	case *ssa.IndexAddr:
		return branch.propagateOnValue(instr)
	case *ssa.Phi:
		return branch.propagateOnValue(instr)
	case *ssa.Call:
		return branch.propagateOnValue(instr)
	case *ssa.MakeMap:
		return branch.propagateOnValue(instr)
	case *ssa.TypeAssert:
		return branch.propagateOnValue(instr)
	case *ssa.ChangeType:
		return branch.propagateOnValue(instr)
	case *ssa.Convert:
		return branch.propagateOnValue(instr)
	case *ssa.Slice:
		return branch.propagateOnValue(instr)
	case *ssa.MakeSlice:
		return branch.propagateOnValue(instr)
	case *ssa.MakeChan:
		return branch.propagateOnValue(instr)
	case *ssa.MakeInterface:
		return branch.propagateOnValue(instr)
	case *ssa.MakeClosure:
		return branch.propagateOnValue(instr)
	case *ssa.Lookup:
		return branch.propagateOnValue(instr)
	case *ssa.Return:
		var newBranches []DefUseBranch
		for _, result := range instr.Results {
			branches := branch.propagateOnValue(result)
			newBranches = append(newBranches, branches...)
		}
		return newBranches
	case *ssa.Store:
		fromValue := branch.valueChain[len(branch.valueChain)-1]
		if fromValue == instr.Val {
			panic("In a def-use chain, when the referrer is Store instruction, the used value is always the Addr. Seems no?")
		}
		branch.refCount--
		return branch.propagateOnValue(instr.Val)
	default:
		panic("TODO: " + instr.String())
	}
}

// propagateOnValue return the instructions that defines the used value in the Value.
func (branch DefUseBranch) propagateOnValue(value ssa.Value) DefUseBranches {
	if _, ok := branch.seenValues[value]; ok {
		branch.end = true
		return []DefUseBranch{branch}
	}
	branch.valueChain = append(branch.valueChain, value)
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
		return branch.propagateOnReferrers(value.Referrers())

	case *ssa.BinOp:
		xBranches := branch.propagateOnValue(value.X)
		yBranches := branch.propagateOnValue(value.Y)
		return append(xBranches, yBranches...)
	case *ssa.Extract:
		return branch.propagateOnValue(value.Tuple)
	case *ssa.UnOp:
		switch value.Op {
		case token.NOT,
			token.SUB,
			token.ARROW,
			token.XOR:
			return branch.propagateOnValue(value.X)
		case token.MUL:
			branch.refCount--
			// refCount reaches 0 means this UnOp pass the Value to the left operand, which no longer continue the def-use chain from
			// the original owner.
			if branch.refCount == 0 {
				branch.end = true
				return []DefUseBranch{branch}
			}
			return branch.propagateOnReferrers(value.Referrers())
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
		return branch.propagateOnReferrers(value.Referrers())
	case *ssa.IndexAddr:
		fromValue := branch.valueChain[len(branch.valueChain)-1]
		if fromValue == value.Index {
			branch.end = true
			return []DefUseBranch{branch}
		}
		branch.refCount++
		return branch.propagateOnValue(value.X)
	case *ssa.Phi:
		var newBranches []DefUseBranch
		for _, edge := range value.Edges {
			branches := branch.propagateOnValue(edge)
			newBranches = append(newBranches, branches...)
		}
		return newBranches
	case *ssa.Call:
		callcomm := value.Common()
		if callcomm.IsInvoke() {
			// invoke mode (dynamic dispatch on interface)
			// TODO: figure out how to get the concrete Call instead of the interface abstract method
			branch.end = true
			return []DefUseBranch{branch}
		} else {
			// call mode
			switch v := callcomm.Value.(type) {
			case *ssa.Function:
				var newBranches []DefUseBranch
				for _, b := range v.Blocks {
					// The return instruction is guaranteed to be the last instruction in each BasicBlock
					if instr, ok := b.Instrs[len(b.Instrs)-1].(*ssa.Return); ok {
						branches := branch.propagateOnInstr(instr) // TODO: Will there be cyclic ref?
						newBranches = append(newBranches, branches...)
					}
					// TODO: if the return value ultimately sourced from parameter, we will need to go on analyzing parameters.
				}
				return newBranches
			case *ssa.MakeClosure:
				panic("TODO:" + value.String())
			case *ssa.Builtin:
				panic("TODO:" + value.String())
			default:
				panic("should not reach here")
			}
		}
	case *ssa.Convert:
		return branch.propagateOnValue(value.X)
	case *ssa.Slice:
		return branch.propagateOnValue(value.X)
	case *ssa.MakeInterface:
		return branch.propagateOnValue(value.X)
	case *ssa.Lookup:
		return branch.propagateOnValue(value.X)
	case *ssa.Global:
		vt := value.Type()
		cnt := 0
		for pt, ok := vt.(*types.Pointer); ok; {
			cnt++
			vt = pt.Elem()
			pt, ok = vt.(*types.Pointer)
		}
		branch.refCount += cnt
		return branch.propagateOnReferrers(value.Referrers())
	case *ssa.Parameter:
		panic("TODO: " + value.String())
	case *ssa.TypeAssert,
		*ssa.ChangeType,
		*ssa.MakeMap,
		*ssa.MakeSlice,
		*ssa.MakeChan,
		*ssa.MakeClosure,
		*ssa.FreeVar,
		*ssa.Const:
		branch.end = true
		return []DefUseBranch{branch}
	default:
		panic("TODO:" + value.String())
	}
}

func (branch DefUseBranch) String() string {
	var valuePos []string
	for _, v := range branch.valueChain {
		valuePos = append(valuePos, branch.fset.Position(v.Pos()).String())
	}
	fields := branch.fields.String()
	return fmt.Sprintf(`%q
	%s`, fields, strings.Join(valuePos, "\n\t")) + "\n"
}
