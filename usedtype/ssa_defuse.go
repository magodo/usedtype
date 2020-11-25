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

type funcParamArg struct {
	param *ssa.Parameter
	arg   ssa.Value
}

type funcParamArgs []funcParamArg

func (pa funcParamArgs) toMap() map[*ssa.Parameter]ssa.Value {
	m := map[*ssa.Parameter]ssa.Value{}
	for _, v := range pa {
		m[v.param] = v.arg
	}
	return m
}

var stagedParamArgKey = new(ssa.Function)

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

	// paramArgs is a map, which mapping each Function Parameter to CallCommon Argument.
	// The purpoes is to allow us relate the parameter with its argument in Function Value procession.
	// (E.g. if a Function Return some parameter, we can figure out what Value is actual returned)
	// The function call related Value has following relationships:
	//
	//                                        +-> Function
	//                                        |      ^ (fn)
	// Go    --+                  (call) 	--+-> MakeClosure
	//         |                  /           |
	// Defer --->  CallCommon  ---            +-> Builtin
	//         |                  \
	// Call  --+                  (invoke) 	-> TODO
	//
	// Other than the Builtin and invoke mode CallCommon, the other paths finally merge to Function.
	//
	// Since CallCommon holds the argument Value, at that stage, we will store the argument in this paramArgs
	// map, under key "stagedParamArgKey" as a staging. Later when we reach to Function Value, we will move
	// the funcParamArgs from the staged key to the exact key of the parent function. Meanwhile, we will complement
	// the funcParamArgs with the parameters.
	// The element will removed from the map when it reaches Return Value as a last step.
	paramArgs map[*ssa.Function]funcParamArgs

	// returnIndexMap is used to remember the Extract Index of the current Value. When we following the def-use chain
	// back to the Function and hit the Return, this is used to guide us to only go on following the impacted return value,
	// rather than following all return values in a multi return function.
	//
	// Since Extract holds the Index Value at that state, we will store the index in this map, under key "stagedParamArgKey" as
	// a staging. Later when we reach to the Function Value, we will move the index from staged key to the exact key of the parent
	// function.
	// At last, when we reach to the Return Value, we will query this index and then remove it as a last step.
	returnIndexMap map[*ssa.Function]int

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
		paramArgs:        map[*ssa.Function]funcParamArgs{},
		returnIndexMap:   map[*ssa.Function]int{},
		fset:             fset,
	}

	return tmpBranch.propagateOnValue(value)
}

// Walk walks the def-use chain in forward for each input branch. In each pass,
// each branch will move one or more step backward in the def-use chain, which might either
// return the branch itself back (means this branch has ended), or return several new
// branches which diverge because of the Value under used is got "defined" in multiple
// places (in this case, it is because the structure/sub-structure's members are defined
// in different places).
func (branches DefUseBranches) Walk() DefUseBranches {
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

	newPas := make(map[*ssa.Function]funcParamArgs, len(branch.paramArgs))
	for f := range branch.paramArgs {
		newPas[f] = make([]funcParamArg, len(branch.paramArgs[f]))
		copy(newPas[f], branch.paramArgs[f])
	}

	newReturnIndexMap := make(map[*ssa.Function]int, len(branch.returnIndexMap))
	for k, v := range branch.returnIndexMap {
		newReturnIndexMap[k] = v
	}

	return DefUseBranch{
		instr:            branch.instr,
		end:              branch.end,
		valueChain:       newValueChain,
		refCount:         branch.refCount,
		fields:           newFields,
		seenInstructions: newSeenInstructions,
		seenValues:       newSeenValues,
		paramArgs:        newPas,
		returnIndexMap:   newReturnIndexMap,
		fset:             branch.fset,
	}
}

func (branch DefUseBranch) propagateOnReferrers(referrers *[]ssa.Instruction) DefUseBranches {
	newBranch := branch.copy()
	var newBranches []DefUseBranch
	if referrers == nil {
		newBranch.end = true
		newBranches = []DefUseBranch{newBranch}
		return newBranches
	}

	for _, instr := range *referrers {
		if branch.instr != nil && branch.instr.Block() != nil {
			// In a def-use chain, we should skip referrer instructions that can not reach to the from instruction
			// Otherwise, we should skip referrer that can not be reached.
			if !branch.instr.Block().Dominates(instr.Block()) {
				continue
			}
		}
		b := newBranch.copy()
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
	newBranch := branch.copy()
	if _, ok := newBranch.seenInstructions[instr]; ok {
		newBranch.end = true
		return []DefUseBranch{newBranch}
	}
	newBranch.seenInstructions[instr] = struct{}{}
	switch instr := instr.(type) {
	case *ssa.Alloc:
		return newBranch.propagateOnValue(instr)
	case *ssa.BinOp:
		return newBranch.propagateOnValue(instr)
	case *ssa.Call:
		return newBranch.propagateOnValue(instr)
	case *ssa.ChangeInterface:
		return newBranch.propagateOnValue(instr)
	case *ssa.ChangeType:
		return newBranch.propagateOnValue(instr)
	case *ssa.Convert:
		return newBranch.propagateOnValue(instr)
	case *ssa.DebugRef:
		newBranch.end = true
		return []DefUseBranch{newBranch}
	case *ssa.Defer:
		return newBranch.propagateOnCallCommon(instr.Call)
	case *ssa.Extract:
		return newBranch.propagateOnValue(instr)
	case *ssa.Field:
		return newBranch.propagateOnValue(instr)
	case *ssa.FieldAddr:
		return newBranch.propagateOnValue(instr)
	case *ssa.Go:
		return newBranch.propagateOnCallCommon(instr.Call)
	case *ssa.If:
		return newBranch.propagateOnValue(instr.Cond)
	case *ssa.Index:
		return newBranch.propagateOnValue(instr)
	case *ssa.IndexAddr:
		return newBranch.propagateOnValue(instr)
	case *ssa.Jump:
		newBranch.end = true
		return []DefUseBranch{newBranch}
	case *ssa.Lookup:
		return newBranch.propagateOnValue(instr)
	case *ssa.MakeChan:
		return newBranch.propagateOnValue(instr)
	case *ssa.MakeClosure:
		return newBranch.propagateOnValue(instr)
	case *ssa.MakeInterface:
		return newBranch.propagateOnValue(instr)
	case *ssa.MakeMap:
		return newBranch.propagateOnValue(instr)
	case *ssa.MakeSlice:
		return newBranch.propagateOnValue(instr)
	case *ssa.MapUpdate:
		newBranch.end = true
		return []DefUseBranch{newBranch}
	case *ssa.Next:
		newBranch.end = true
		return []DefUseBranch{newBranch}
	case *ssa.Panic:
		newBranch.end = true
		return []DefUseBranch{newBranch}
	case *ssa.Phi:
		return newBranch.propagateOnValue(instr)
	case *ssa.Range:
		return newBranch.propagateOnValue(instr)
	case *ssa.Return:
		var result ssa.Value
		switch len(instr.Results) {
		case 0:
			newBranch.end = true
			return []DefUseBranch{newBranch}
		case 1:
			result = instr.Results[0]
		default:
			result = instr.Results[newBranch.returnIndexMap[instr.Parent()]]
		}

		// TODO: following is WRONG, we should construct a use-def chain and flow back to the def point.
		newBranches := newBranch.propagateOnValue(result)

		// Look for the def Value of the target result, and then start from there.
		// In case the referrer is nil, i.e. the result is one of:
		// - named Functions
		// - Builtin
		// - Const
		// - Global
		// Then we will simply go on propagating those value.
		//var newBranches []DefUseBranch
		//preferrers := result.Referrers()
		//if preferrers == nil {
		//	newBranches = newBranch.propagateOnValue(result)
		//} else {
		//	// TODO: construct a use-def chain and flow back to the def point.
		//}

		// remove the paramArgs for current function
		for _, b := range newBranches {
			delete(b.paramArgs, instr.Parent())
		}
		// remove the index for current function
		delete(newBranch.returnIndexMap, instr.Parent())

		return newBranches
	case *ssa.RunDefers:
		newBranch.end = true
		return []DefUseBranch{newBranch}
	case *ssa.Select:
		return newBranch.propagateOnValue(instr)
	case *ssa.Send:
		return newBranch.propagateOnValue(instr.X)
	case *ssa.Slice:
		return newBranch.propagateOnValue(instr)
	case *ssa.Store:
		fromValue := newBranch.valueChain[len(newBranch.valueChain)-1]
		if fromValue == instr.Val {
			panic("In a def-use chain, when the referrer is Store instruction, the used value is always the Addr")
		}
		newBranch.refCount--
		return newBranch.propagateOnValue(instr.Val)
	case *ssa.TypeAssert:
		return newBranch.propagateOnValue(instr)
	case *ssa.UnOp:
		return newBranch.propagateOnValue(instr)
	default:
		panic("Not gonna happen")
	}
}

// propagateOnValue return the instructions that defines the used value in the Value.
func (branch DefUseBranch) propagateOnValue(value ssa.Value) DefUseBranches {
	newBranch := branch.copy()
	if _, ok := newBranch.seenValues[value]; ok {
		newBranch.end = true
		return []DefUseBranch{newBranch}
	}
	newBranch.valueChain = append(newBranch.valueChain, value)
	newBranch.seenValues[value] = struct{}{}

	switch value := value.(type) {
	case *ssa.Alloc:
		vt := value.Type()
		cnt := 0
		for pt, ok := vt.(*types.Pointer); ok; {
			cnt++
			vt = pt.Elem()
			pt, ok = vt.(*types.Pointer)
		}
		newBranch.refCount += cnt
		return newBranch.propagateOnReferrers(value.Referrers())

	case *ssa.BinOp:
		xBranches := newBranch.propagateOnValue(value.X)
		yBranches := newBranch.propagateOnValue(value.Y)
		return append(xBranches, yBranches...)
	case *ssa.Builtin:
		// TODO
		newBranch.end = true
		return []DefUseBranch{newBranch}
	case *ssa.Call:
		callcomm := value.Common()
		if callcomm == nil {
			newBranch.end = true
			return []DefUseBranch{newBranch}
		}
		return newBranch.propagateOnCallCommon(*callcomm)
	case *ssa.ChangeInterface:
		newBranch.end = true
		return []DefUseBranch{newBranch}
	case *ssa.ChangeType:
		newBranch.end = true
		return []DefUseBranch{newBranch}
	case *ssa.Const:
		newBranch.end = true
		return []DefUseBranch{newBranch}
	case *ssa.Convert:
		return newBranch.propagateOnValue(value.X)
	case *ssa.Extract:
		branch.returnIndexMap[stagedParamArgKey] = value.Index
		return newBranch.propagateOnValue(value.Tuple)
	case *ssa.Field:
		return newBranch.propagateOnValue(value.X)
	case *ssa.FieldAddr:
		vt := value.X.Type().(*types.Pointer).Elem().Underlying().(*types.Struct).Field(value.Field).Type()
		cnt := 1
		for pt, ok := vt.(*types.Pointer); ok; {
			cnt++
			vt = pt.Elem()
			pt, ok = vt.(*types.Pointer)
		}
		newBranch.refCount = cnt
		return newBranch.propagateOnReferrers(value.Referrers())
	case *ssa.FreeVar:
		newBranch.end = true
		return []DefUseBranch{newBranch}
	case *ssa.Function:
		// Move the staged paramArg from the staged place to the exact function place,
		// then complement the paramArg map for the parameter part
		pas := newBranch.paramArgs[stagedParamArgKey]
		delete(newBranch.paramArgs, stagedParamArgKey)
		for i := range pas {
			pas[i].param = value.Params[i]
		}
		newBranch.paramArgs[value] = pas

		// Move the staged index from the staged place to the exact function place.
		newBranch.returnIndexMap[value] = newBranch.returnIndexMap[stagedParamArgKey]
		delete(newBranch.returnIndexMap, stagedParamArgKey)

		var newBranches []DefUseBranch

		var param ssa.Value
		for _, pa := range pas {
			for _, fromValue := range newBranch.valueChain {
				if pa.arg == fromValue {
					param = pa.param
					break
				}
			}
		}
		if param != nil {
			// If the fromValue is in one of the arguments, it means that Value could be mutate in this function.
			return newBranch.propagateOnValue(param)

		} else {
			// Otherwise it means the fromValue is the return value, then we will start from Return Value
			for _, b := range value.Blocks {
				// The return instruction is guaranteed to be the last instruction in each BasicBlock
				if instr, ok := b.Instrs[len(b.Instrs)-1].(*ssa.Return); ok {
					branches := newBranch.propagateOnInstr(instr)
					newBranches = append(newBranches, branches...)
				}
			}
		}
		return newBranches
	case *ssa.Global:
		vt := value.Type()
		cnt := 0
		for pt, ok := vt.(*types.Pointer); ok; {
			cnt++
			vt = pt.Elem()
			pt, ok = vt.(*types.Pointer)
		}
		newBranch.refCount += cnt
		return newBranch.propagateOnReferrers(value.Referrers())
	case *ssa.Index:
		newBranch.end = true
		return []DefUseBranch{newBranch}
	case *ssa.IndexAddr:
		newBranch.end = true
		return []DefUseBranch{newBranch}
	case *ssa.Lookup:
		return newBranch.propagateOnValue(value.X)
	case *ssa.MakeChan:
		newBranch.end = true
		return []DefUseBranch{newBranch}
	case *ssa.MakeInterface:
		return newBranch.propagateOnValue(value.X)
	case *ssa.MakeMap:
		newBranch.end = true
		return []DefUseBranch{newBranch}
	case *ssa.MakeSlice:
		newBranch.end = true
		return []DefUseBranch{newBranch}
	case *ssa.MakeClosure:
		newBranch.end = true
		return []DefUseBranch{newBranch}
	case *ssa.Next:
		newBranch.end = true
		return []DefUseBranch{newBranch}
	case *ssa.Parameter:
		return newBranch.propagateOnReferrers(value.Referrers())
	case *ssa.Phi:
		var newBranches []DefUseBranch
		for _, edge := range value.Edges {
			branches := newBranch.propagateOnValue(edge)
			newBranches = append(newBranches, branches...)
		}
		return newBranches
	case *ssa.Range:
		newBranch.end = true
		return []DefUseBranch{newBranch}
	case *ssa.Select:
		var newBranches []DefUseBranch
		for _, instr := range value.Block().Instrs {
			branches := newBranch.propagateOnInstr(instr)
			newBranches = append(newBranches, branches...)
		}
		return newBranches
	case *ssa.Slice:
		newBranch.end = true
		return []DefUseBranch{newBranch}
	case *ssa.TypeAssert:
		newBranch.end = true
		return []DefUseBranch{newBranch}
	case *ssa.UnOp:
		switch value.Op {
		case token.NOT,
			token.SUB,
			token.ARROW,
			token.XOR:
			return newBranch.propagateOnValue(value.X)
		case token.MUL:
			newBranch.refCount--
			// refCount reaches 0 means this UnOp pass the Value to the left operand, which no longer continue the def-use chain from
			// the original owner.
			if newBranch.refCount == 0 {
				newBranch.end = true
				return []DefUseBranch{newBranch}
			}
			return newBranch.propagateOnReferrers(value.Referrers())
		default:
			panic("Not gonna happen")
		}
	default:
		panic("Not gonna happen")
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

func (branch DefUseBranch) propagateOnCallCommon(callcomm ssa.CallCommon) DefUseBranches {
	if callcomm.IsInvoke() {
		// invoke mode (dynamic dispatch on interface)
		// TODO: figure out how to get the concrete Call instead of the interface abstract method
		branch.end = true
		return []DefUseBranch{branch.copy()}
	}
	// call mode

	// Initialize the paramArgs, but put it in a stage place. This will be later moved to
	// the corresponding place (keyed by the function *ssa.Function), when we hit the Function Value.
	pas := make([]funcParamArg, len(callcomm.Args))
	for i := range pas {
		pas[i].arg = callcomm.Args[i]
	}
	branch.paramArgs[stagedParamArgKey] = pas
	return branch.propagateOnValue(callcomm.Value)
}
