package usedtype

import (
	"fmt"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/ssa"
)

type chainState int

const (
	chainActive chainState = iota
	chainEndNoUse
	chainEndProvider
	chainEndConsumer
	chainEndCyclicUse
	chainEndOutOfBoundary
)

type ODUChain struct {
	// pkgBoundary is the package boundary of our check. When propagating on function, we will avoid crossing the boundary.
	pkgBoundary map[*ssa.Package]struct{}

	// the starting value (def value) of this chain
	root ssa.Value

	// refCount is used to keep track how many times current Value is referenced (&).
	// This will be added each time it is referenced (&), and will be reduced each time
	// it is de-referenced (*).
	refCount int

	// instrChain keep track of the def-use instruction chain starting from the firs instruction.
	instrChain []ssa.Instruction

	// valueChain keep track of the def-use value chain starting from the def value.
	valueChain []ssa.Value

	// fields keep any struct field in the def-use chain
	fields structFields

	// seenInstructions keep all the instructions met till now, to avoid cyclic reference
	seenInstructions map[ssa.Instruction]struct{}

	// seenValues keep all the Values met till now, to avoid cyclic reference
	seenValues map[ssa.Value]struct{}

	// state of the chain
	state chainState

	// funcArgs keep track of which argument is used in the CallCommon, which will be later used in the Function Value
	// to determine to follow which parameter
	argIndex int

	// callPointLookup is used as a lookup table to find Referrers to function.
	callPointLookup *CallPointLookup

	// returnIndex is used to remember the Extract Index of the current Value.
	// This is set in instructions which will return Extract:
	// - Call
	// - TypeAssert
	// - Next
	// - UnOp
	returnIndex int

	// for debug purpose only
	fset *token.FileSet
}

type ODUChains []ODUChain

func WalkODUChains(value ssa.Value, pkgs []*ssa.Package, fset *token.FileSet) ODUChains {
	if !IsDefValue(value) {
		panic(`value is not "def" value`)
	}

	pkgBoundary := map[*ssa.Package]struct{}{}
	for _, pkg := range pkgs {
		pkgBoundary[pkg] = struct{}{}
	}

	chain := ODUChain{
		pkgBoundary:      pkgBoundary,
		root:             value,
		refCount:         ReferenceDepth(value.Type()),
		instrChain:       []ssa.Instruction{},
		valueChain:       []ssa.Value{value}, // record root value
		fields:           []structField{},
		seenInstructions: map[ssa.Instruction]struct{}{},
		seenValues:       map[ssa.Value]struct{}{value: {}}, // record root value
		callPointLookup:  callPointLookup,
		state:            chainActive,
		fset:             fset,
	}
	return chain.propagateOnReferrers(value.Referrers())
}

func (ochain ODUChain) copy() ODUChain {
	newPkgBoundary := make(map[*ssa.Package]struct{}, len(ochain.pkgBoundary))
	for k, v := range ochain.pkgBoundary {
		newPkgBoundary[k] = v
	}

	newInstrChain := make([]ssa.Instruction, len(ochain.instrChain))
	copy(newInstrChain, ochain.instrChain)

	newValueChain := make([]ssa.Value, len(ochain.valueChain))
	copy(newValueChain, ochain.valueChain)

	newFields := make([]structField, len(ochain.fields))
	copy(newFields, ochain.fields)

	newSeenInstructions := map[ssa.Instruction]struct{}{}
	for k, v := range ochain.seenInstructions {
		newSeenInstructions[k] = v
	}

	newSeenValues := map[ssa.Value]struct{}{}
	for k, v := range ochain.seenValues {
		newSeenValues[k] = v
	}

	return ODUChain{
		pkgBoundary:      newPkgBoundary,
		root:             ochain.root,
		state:            ochain.state,
		instrChain:       ochain.instrChain,
		valueChain:       newValueChain,
		refCount:         ochain.refCount,
		fields:           newFields,
		seenInstructions: newSeenInstructions,
		seenValues:       newSeenValues,
		argIndex:         ochain.argIndex,
		returnIndex:      ochain.returnIndex,
		callPointLookup:  ochain.callPointLookup,
		fset:             ochain.fset,
	}
}

func (ochain ODUChain) propagateOnReferrers(referrers *[]ssa.Instruction) ODUChains {
	if referrers == nil || len(*referrers) == 0 {
		ochain.state = chainEndNoUse
		return []ODUChain{ochain}
	}

	var newChains []ODUChain
	for _, instr := range *referrers {
		newChains = append(newChains, ochain.propagateOnInstr(instr)...)
	}

	return newChains
}

// propagateOnInstr return the instructions that defines the used value in the instruction.
func (ochain ODUChain) propagateOnInstr(instr ssa.Instruction) ODUChains {
	chain := ochain.copy()
	if _, ok := chain.seenInstructions[instr]; ok {
		chain.state = chainEndCyclicUse
		return []ODUChain{chain}
	}
	chain.instrChain = append(chain.instrChain, instr)
	chain.seenInstructions[instr] = struct{}{}

	switch instr := instr.(type) {
	case *ssa.Alloc:
		// def-use ochain will not reach this point
		panic("should never happen")
	case *ssa.BinOp:
		return chain.propagateOnValue(instr)
	case *ssa.Call:
		return chain.propagateOnCallCommon(instr.Call)
	case *ssa.ChangeInterface:
		return chain.propagateOnValue(instr)
	case *ssa.ChangeType:
		return chain.propagateOnValue(instr)
	case *ssa.Convert:
		return chain.propagateOnValue(instr)
	case *ssa.DebugRef:
		chain.state = chainEndNoUse
		return []ODUChain{chain}
	case *ssa.Defer:
		return chain.propagateOnCallCommon(instr.Call)
	case *ssa.Extract:
		if instr.Index != chain.returnIndex {
			chain.state = chainEndNoUse
			return []ODUChain{chain}
		}
		chain.returnIndex = 0 // reset
		return chain.propagateOnValue(instr)
	case *ssa.Field:
		chain.fields = append(chain.fields,
			structField{
				index: instr.Field,
				t:     instr.X.Type(),
			})
		return chain.propagateOnValue(instr)
	case *ssa.FieldAddr:
		chain.fields = append(chain.fields,
			structField{
				index: instr.Field,
				t:     instr.X.Type(),
			})

		// We are going to track the ownership of a new object from this point,
		// hence we should reset the refCount.
		chain.refCount = ReferenceDepth(instr.X.Type().(*types.Pointer).Elem().Underlying().(*types.Struct).Field(instr.Field).Type()) + 1
		return chain.propagateOnValue(instr)
	case *ssa.Go:
		return chain.propagateOnCallCommon(instr.Call)
	case *ssa.If:
		chain.state = chainEndNoUse
		return []ODUChain{chain}
	case *ssa.Index:
		fromValue := chain.valueChain[len(chain.valueChain)-1]
		if fromValue == instr.Index {
			chain.state = chainEndNoUse
			return []ODUChain{chain}
		}
		return chain.propagateOnValue(instr)
	case *ssa.IndexAddr:
		fromValue := chain.valueChain[len(chain.valueChain)-1]
		if fromValue == instr.Index {
			chain.state = chainEndNoUse
			return []ODUChain{chain}
		}
		chain.refCount = ReferenceDepth(instr.X.Type()) + 1
		return chain.propagateOnValue(instr)
	case *ssa.Jump:
		panic("should never happen")
	case *ssa.Lookup:
		fromValue := chain.valueChain[len(chain.valueChain)-1]
		if fromValue == instr.Index {
			chain.state = chainEndNoUse
			return []ODUChain{chain}
		}
		return chain.propagateOnValue(instr)
	case *ssa.MakeChan:
		chain.state = chainEndNoUse
		return []ODUChain{chain}
	case *ssa.MakeClosure:
		panic("TODO")
	case *ssa.MakeInterface:
		return chain.propagateOnValue(instr)
	case *ssa.MakeMap:
		chain.state = chainEndNoUse
		return []ODUChain{chain}
	case *ssa.MakeSlice:
		chain.state = chainEndNoUse
		return []ODUChain{chain}
	case *ssa.MapUpdate:
		chain.state = chainEndNoUse
		return []ODUChain{chain}
	case *ssa.Next:
		chain.returnIndex = 2 // (ok, k, v)
		return chain.propagateOnValue(instr)
	case *ssa.Panic:
		chain.state = chainEndNoUse
		return []ODUChain{chain}
	case *ssa.Phi:
		return chain.propagateOnValue(instr)
	case *ssa.Range:
		return chain.propagateOnValue(instr)
	case *ssa.Return:
		fromValue := chain.valueChain[len(chain.valueChain)-1]
		var idx = -1
		for i, result := range instr.Results {
			if result == fromValue {
				idx = i
				break
			}
		}
		assert(idx >= 0)
		chain.returnIndex = idx

		// Find the call point on the including function, which are the referrers to this instruction (return of a function)
		referrers := chain.callPointLookup.FindCallPoint(instr.Parent().Pkg)[instr.Parent()]
		if referrers == nil {
			chain.state = chainEndNoUse
			return []ODUChain{chain}
		}
		// Here each referrer is referring to the enclosing function, which is (e.g.) a Call, Defer, etc.
		// If the referrer itself is a Value, we will go on following its referrers.
		var newChains []ODUChain
		for _, referrer := range referrers {
			if value, ok := referrer.(ssa.Value); ok {
				chain := chain.copy()
				if _, ok := chain.seenValues[value]; ok {
					chain.state = chainEndCyclicUse
					newChains = append(newChains, chain)
					continue
				}
				chain.valueChain = append(chain.valueChain, value)
				chain.seenValues[value] = struct{}{}
				newChains = append(newChains, chain.propagateOnValue(value)...)
			}
		}
		// In case all the referrers are instruction only, we will end this chain.
		if len(newChains) == 0 {
			chain.state = chainEndNoUse
			return []ODUChain{chain}
		}
		return newChains
	case *ssa.RunDefers:
		chain.state = chainEndNoUse
		return []ODUChain{chain}
	case *ssa.Select:
		panic("TODO")
	case *ssa.Send:
		chain.state = chainEndNoUse
		return []ODUChain{chain}
	case *ssa.Slice:
		fromValue := chain.valueChain[len(chain.valueChain)-1]
		if fromValue != instr.X {
			chain.state = chainEndNoUse
			return []ODUChain{chain}
		}
		return chain.propagateOnValue(instr)
	case *ssa.Store:
		fromValue := chain.valueChain[len(chain.valueChain)-1]
		if fromValue == instr.Val {
			chain.state = chainEndProvider
		}
		chain.state = chainEndConsumer
		return []ODUChain{chain}
	case *ssa.TypeAssert:
		if !instr.CommaOk {
			chain.returnIndex = 0
		}
		return chain.propagateOnValue(instr)
	case *ssa.UnOp:
		if instr.CommaOk && instr.Op == token.ARROW {
			chain.returnIndex = 0
		}
		return chain.propagateOnValue(instr)
	default:
		panic("Not gonna happen")
	}
}

// propagateOnValue return the instructions that defines the used value in the Value.
func (ochain ODUChain) propagateOnValue(value ssa.Value) ODUChains {
	chain := ochain.copy()
	if _, ok := chain.seenValues[value]; ok {
		chain.state = chainEndCyclicUse
		return []ODUChain{chain}
	}
	chain.valueChain = append(chain.valueChain, value)
	chain.seenValues[value] = struct{}{}

	switch value := value.(type) {
	case *ssa.Alloc:
		// def-use ochain will not reach this point
		panic("will never happen")
	case *ssa.BinOp:
		return chain.propagateOnReferrers(value.Referrers())
	case *ssa.Builtin:
		chain.state = chainEndNoUse
		return []ODUChain{chain}
	case *ssa.Call:
		// this is handled in the instruction level
		panic("will never happen")
	case *ssa.ChangeInterface:
		return chain.propagateOnReferrers(value.Referrers())
	case *ssa.ChangeType:
		return chain.propagateOnReferrers(value.Referrers())
	case *ssa.Const:
		chain.state = chainEndNoUse
		return []ODUChain{chain}
	case *ssa.Convert:
		return chain.propagateOnReferrers(value.Referrers())
	case *ssa.Extract:
		return chain.propagateOnReferrers(value.Referrers())
	case *ssa.Field:
		return chain.propagateOnReferrers(value.Referrers())
	case *ssa.FieldAddr:
		return chain.propagateOnReferrers(value.Referrers())
	case *ssa.FreeVar:
		panic("TODO")
	case *ssa.Function:
		// Only keep the ochain in the boundary of current package
		if value.Package() != nil {
			if _, ok := chain.pkgBoundary[value.Package()]; !ok {
				chain.state = chainEndOutOfBoundary
				return []ODUChain{chain}
			}
		}
		param := value.Params[chain.argIndex]
		chain.argIndex = 0 // reset
		return chain.propagateOnValue(param)
	case *ssa.Global:
		// def-use ochain will not reach this point
		panic("will never happen")
	case *ssa.Index:
		return chain.propagateOnReferrers(value.Referrers())
	case *ssa.IndexAddr:
		return chain.propagateOnReferrers(value.Referrers())
	case *ssa.Lookup:
		return chain.propagateOnReferrers(value.Referrers())
	case *ssa.MakeChan:
		panic("will never happen")
	case *ssa.MakeInterface:
		return chain.propagateOnReferrers(value.Referrers())
	case *ssa.MakeMap:
		panic("will never happen")
	case *ssa.MakeSlice:
		panic("will never happen")
	case *ssa.MakeClosure:
		panic("TODO")
	case *ssa.Next:
		return chain.propagateOnReferrers(value.Referrers())
	case *ssa.Parameter:
		return chain.propagateOnReferrers(value.Referrers())
	case *ssa.Phi:
		return chain.propagateOnReferrers(value.Referrers())
	case *ssa.Range:
		return chain.propagateOnReferrers(value.Referrers())
	case *ssa.Select:
		panic("TODO")
	case *ssa.Slice:
		return chain.propagateOnReferrers(value.Referrers())
	case *ssa.TypeAssert:
		return chain.propagateOnReferrers(value.Referrers())
	case *ssa.UnOp:
		switch value.Op {
		case token.ARROW:
			panic("TODO")
		case token.NOT,
			token.SUB,
			token.XOR:
			return chain.propagateOnReferrers(value.Referrers())
		case token.MUL:
			chain.refCount--
			assert(chain.refCount >= 0)
			return chain.propagateOnReferrers(value.Referrers())
		default:
			panic("will never happen")
		}
	default:
		panic("will never happen")
	}
}

func (ochain ODUChain) propagateOnCallCommon(com ssa.CallCommon) ODUChains {
	fromValue := ochain.valueChain[len(ochain.valueChain)-1]
	var index = -1
	for i := range com.Args {
		if com.Args[i] == fromValue {
			index = i
			break
		}
	}
	assert(index >= 0)
	ochain.argIndex = index

	if com.IsInvoke() {
		// invoke mode (dynamic dispatch on interface)
		panic("TODO")
	}
	return ochain.propagateOnValue(com.Value)
}

func (ochain ODUChain) String() string {
	var positions []string

	for _, instr := range ochain.instrChain {
		suffix := ""
		if _, ok := instr.(*ssa.Phi); ok {
			suffix = " (phi)"
		}
		positions = append(positions, ochain.fset.Position(instr.Pos()).String()+suffix)
	}
	fields := ochain.fields.String()
	return fmt.Sprintf(`%s (%s): %q
	%s`, ochain.fset.Position(ochain.root.Pos()).String(), ochain.root, fields, strings.Join(positions, "\n\t"))
}
