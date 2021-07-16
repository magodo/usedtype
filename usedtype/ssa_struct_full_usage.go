package usedtype

import (
	"go/types"
	"sort"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"

	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/ssa"
)

var verbose bool

func SetStructFieldUsageVerbose(enabled bool) {
	verbose = enabled
}

type StructFullUsageKey struct {
	Named   *types.Named
	Variant *types.Named // non-nil only when Named is a Named interface_property
}

type StructFullUsageKeys []StructFullUsageKey

type StructFieldFullUsageKey struct {
	StructField
	Variant *types.Named // non-nil only when the StructField corresponds to an interface_property
}

type StructFieldFullUsageKeys []StructFieldFullUsageKey

type StructFieldFullUsage struct {
	Key              StructFieldFullUsageKey
	NestedFields     StructNestedFields
	VirtAccessPoints map[VirtAccessPoint]struct{}

	dm             StructDirectUsageMap
	seenStructures map[*types.Named]struct{}
}

type StructNestedFields map[StructFieldFullUsageKey]StructFieldFullUsage

type StructFullUsage struct {
	Key          StructFullUsageKey
	Alloc        Alloc
	NestedFields StructNestedFields

	dm StructDirectUsageMap
}

type StructFullUsageAmongAlloc map[Alloc]StructFullUsage

type StructFullUsages struct {
	dm               StructDirectUsageMap
	UsagesAmongAlloc map[StructFullUsageKey]StructFullUsageAmongAlloc
}

func (keys StructFullUsageKeys) Len() int {
	return len(keys)
}
func (keys StructFullUsageKeys) Swap(i, j int) {
	keys[i], keys[j] = keys[j], keys[i]
}

func (keys StructFullUsageKeys) Less(i, j int) bool {
	return keys[i].Named.String() < keys[j].Named.String() ||
		(keys[i].Named.String() == keys[j].Named.String() &&
			keys[i].Variant.String() < keys[j].Variant.String())
}

func (keys StructFieldFullUsageKeys) Len() int {
	return len(keys)
}
func (keys StructFieldFullUsageKeys) Swap(i, j int) {
	keys[i], keys[j] = keys[j], keys[i]
}

func (keys StructFieldFullUsageKeys) Less(i, j int) bool {
	return keys[i].index < keys[j].index ||
		(keys[i].index == keys[j].index &&
			keys[i].Variant.String() < keys[j].Variant.String())
}

func (key StructFullUsageKey) String() string {
	if key.Variant == nil {
		return key.Named.String()
	}
	return key.Named.String() + " [" + key.Variant.String() + "]"
}

func (key StructFieldFullUsageKey) String() string {
	if key.Variant == nil {
		return key.StructField.String()
	}
	return key.StructField.String() + " [" + key.Variant.String() + "]"
}

func (fu StructFullUsage) String() string {
	var out = []string{fu.Key.String()}
	if verbose {
		out = append(out, fu.Alloc.Position.String())
	}

	var keys StructFieldFullUsageKeys = make([]StructFieldFullUsageKey, len(fu.NestedFields))
	cnt := 0
	for k := range fu.NestedFields {
		keys[cnt] = k
		cnt++
	}
	sort.Sort(keys)

	for _, key := range keys {
		out = append(out, fu.NestedFields[key].stringWithIndent(2))
	}
	return strings.Join(out, "\n")
}

func (fus StructFullUsages) String() string {
	keys := make(StructFullUsageKeys, 0, len(fus.UsagesAmongAlloc))
	for k := range fus.UsagesAmongAlloc {
		keys = append(keys, k)
	}
	sort.Sort(keys)

	var out []string
	for _, key := range keys {
		usageAmongAlloc := fus.UsagesAmongAlloc[key]

		if !verbose {
			fu := usageAmongAlloc.Flatten()
			if fu == nil {
				continue
			}
			out = append(out, fu.String())
			continue
		}

		// In verbose mode, we will print all the instances of a struct full usage
		allocs := make(Allocs, 0, len(usageAmongAlloc))
		for alloc := range usageAmongAlloc {
			allocs = append(allocs, alloc)
		}
		sort.Sort(allocs)

		for _, alloc := range allocs {
			out = append(out, usageAmongAlloc[alloc].String())
		}
	}
	return strings.Join(out, "\n")
}

func (ffu StructFieldFullUsage) String() string {
	return ffu.stringWithIndent(0)
}

func (ffu StructFieldFullUsage) stringWithIndent(ident int) string {
	prefix := strings.Repeat("  ", ident)
	var out = []string{prefix + ffu.Key.String()}

	if verbose {
		positions := make([]string, 0, len(ffu.VirtAccessPoints))
		for vnode := range ffu.VirtAccessPoints {
			positions = append(positions, prefix+"  "+vnode.Pos.String())
		}
		out = append(out, positions...)
	}

	var keys StructFieldFullUsageKeys = make([]StructFieldFullUsageKey, len(ffu.NestedFields))
	cnt := 0
	for k := range ffu.NestedFields {
		keys[cnt] = k
		cnt++
	}
	sort.Sort(keys)

	for _, key := range keys {
		out = append(out, ffu.NestedFields[key].stringWithIndent(ident+2))
	}
	return strings.Join(out, "\n")
}

func (ffu StructFieldFullUsage) copy() StructFieldFullUsage {
	newNestedFields := make(map[StructFieldFullUsageKey]StructFieldFullUsage)
	for k, v := range ffu.NestedFields {
		newNestedFields[k] = v.copy()
	}

	newSeenStructs := make(map[*types.Named]struct{})
	for k, v := range ffu.seenStructures {
		newSeenStructs[k] = v
	}

	newPoints := make(map[VirtAccessPoint]struct{}, len(ffu.VirtAccessPoints))
	for k, v := range ffu.VirtAccessPoints {
		newPoints[k] = v
	}

	return StructFieldFullUsage{
		dm:               ffu.dm,
		Key:              ffu.Key,
		NestedFields:     newNestedFields,
		seenStructures:   newSeenStructs,
		VirtAccessPoints: newPoints,
	}
}

// build build nested fields for a given Named structure or Named interface (baseStruct).
func (nsf StructNestedFields) build(dm StructDirectUsageMap, baseStruct *types.Named, seenStructures map[*types.Named]struct{}, origin Alloc, opt *StructFullBuildOption) {
	if _, ok := seenStructures[baseStruct]; ok {
		return
	}
	seenStructures[baseStruct] = struct{}{}

	du, ok := dm[baseStruct]
	if !ok {
		return
	}

	for nestedField, vaps := range du {
		nestedFieldType := nestedField.DereferenceRElem()
		vAccessPoints := make(map[VirtAccessPoint]struct{})

		// Check whether this virtual access can be tracked from the original virtual access point
		for _, vap := range vaps {
			if opt != nil && opt.Callgraph != nil {
				if !checkInstructionReachability(origin.Instr, vap.Instr, opt.Callgraph) {
					continue
				}
			}
			vAccessPoints[vap] = struct{}{}
			// In non-verbose mode, there is no need to record all vaps, only one is enough.
			if !verbose {
				break
			}
		}

		if len(vAccessPoints) == 0 {
			continue
		}

		ffu := StructFieldFullUsage{
			dm:               dm,
			seenStructures:   seenStructures,
			NestedFields:     map[StructFieldFullUsageKey]StructFieldFullUsage{},
			VirtAccessPoints: vAccessPoints,
		}

		if !IsElemUnderlyingNamedStructOrInterface(nestedFieldType) {
			k := StructFieldFullUsageKey{
				StructField: nestedField,
			}
			ffu.Key = k
			nsf[k] = ffu
			continue
		}

		nt := nestedFieldType.(*types.Named)
		switch t := nt.Underlying().(type) {
		case *types.Interface:
			for du := range dm {
				if opt != nil && opt.CustomImplements != nil {
					if !opt.CustomImplements(du, nt) {
						continue
					}
				} else {
					if !types.Implements(du, t) {
						continue
					}
				}
				ffu := ffu.copy()
				k := StructFieldFullUsageKey{
					StructField: nestedField,
					Variant:     du,
				}
				ffu.Key = k
				ffu.NestedFields.build(dm, du, ffu.seenStructures, origin, opt)
				nsf[k] = ffu
			}
		case *types.Struct:
			ffu := ffu.copy()
			k := StructFieldFullUsageKey{
				StructField: nestedField,
			}
			ffu.Key = k
			ffu.NestedFields.build(dm, nt, ffu.seenStructures, origin, opt)
			nsf[k] = ffu
		default:
			panic("will never happen")
		}
	}
}

// checkInstructionReachability checks whether two instructions can reach the other in either direction.
// Ideally, for a read field access, we should ensure the root structure can reach the child field's read;
// Otherwise, for a write field access, we should ensure the write of the child field happens first.
// However, it is non-trivial in SSA to determine whether one instruction (Field/FieldAddr) is for a later read or write.
// Practically, we ignore this difference here, but simply check whether two instructions can reach the other in either direction.
func checkInstructionReachability(i1, i2 ssa.Instruction, graph *callgraph.Graph) bool {
	if i1.Block() == i2.Block() {
		return true
	}
	if i1.Parent() == i2.Parent() {
		i1CanReachI2, i2CanReachI1 := BBCanReach(i1.Block(), i2.Block()), BBCanReach(i2.Block(), i1.Block())
		return i1CanReachI2 || i2CanReachI1
	}

	// In case n1 can reach n2, it only means the function enclosing i1 has at least one
	// path that calls the function enclosing i2.
	// TODO: we should ensure the i1 can reach the callsite. We didn't do it here for now
	// since `callgraph.PathSearch` only returns an arbitrary path, whilst we should check
	// all possible paths.
	n1 := graph.Nodes[i1.Parent()]
	n2 := graph.Nodes[i2.Parent()]

	// For some whole program algorithms (e.g. rta), the callgraph only contains the subset of functions that reachable from main().
	// If the instruction here isn't reachable from main, then we should regard them as not reachable.
	if n1 == nil || n2 == nil {
		return false
	}

	paths1to2 := callgraph.PathSearch(n1, func(n *callgraph.Node) bool {
		return n == n2
	})
	paths2to1 := callgraph.PathSearch(n2, func(n *callgraph.Node) bool {
		return n == n1
	})
	return len(paths1to2) != 0 || len(paths2to1) != 0
}

// buildUsagesAmongAlloc build usages for one Named type, which is either a structure or an interface. In case of interface, it will
// build for all its implementors.
// The meaning of "build usages" here means to regard the input type as the root structure, recursively iterate its fields to
// check whether the virtual access from this type to this field occurs in the direct usage map.
func (us StructFullUsages) buildUsagesAmongAlloc(wg *sync.WaitGroup, root *types.Named, allocSet AllocSet, opt *StructFullBuildOption) {
	// If the target Named type is an interface_property, we shall do the full usage processing
	// on each of its variants that appear in the direct usage map.
	if iRoot, ok := root.Underlying().(*types.Interface); ok {
		for named := range us.dm {
			if opt != nil && opt.CustomImplements != nil {
				if !opt.CustomImplements(named, root) {
					continue
				}
			} else {
				if !types.Implements(named, iRoot) {
					continue
				}
			}
			k := StructFullUsageKey{
				Named:   root,
				Variant: named,
			}
			us.buildUsagesAmongAllocForStructure(wg, k, allocSet, named, opt)
		}
		return
	}

	if _, ok := us.dm[root]; !ok {
		return
	}

	k := StructFullUsageKey{
		Named: root,
	}
	us.buildUsagesAmongAllocForStructure(wg, k, allocSet, root, opt)
	return
}

func (us StructFullUsages) buildUsagesAmongAllocForStructure(wg *sync.WaitGroup, k StructFullUsageKey, allocSet AllocSet, named *types.Named, opt *StructFullBuildOption) {
	usageAmongAlloc := StructFullUsageAmongAlloc{}
	us.UsagesAmongAlloc[k] = usageAmongAlloc
	wg.Add(1)
	go func() {
		defer wg.Done()
		for alloc := range allocSet {
			fu := StructFullUsage{
				dm:           us.dm,
				Key:          k,
				Alloc:        alloc,
				NestedFields: map[StructFieldFullUsageKey]StructFieldFullUsage{},
			}
			usageAmongAlloc[alloc] = fu
			fu.NestedFields.build(us.dm, named, map[*types.Named]struct{}{}, alloc, opt)
		}
		log.Debugf("finish %s\n", named.String())
	}()
}

// Flatten merges all instances of StructFullUsage of a struct appear in different Alloc into one.
// The returned StructFullUsage only has Key and NestedFields filled. Hence it will not show verbose information even
// if verbose is enabled.
func (amongAlloc StructFullUsageAmongAlloc) Flatten() *StructFullUsage {
	var out *StructFullUsage
	for _, fu := range amongAlloc {
		out = &StructFullUsage{
			Key:          fu.Key,
			Alloc:        fu.Alloc,
			NestedFields: StructNestedFields{},
		}
		break
	}
	if out == nil {
		return nil
	}

	// Flatten a field full usage into a StructNestedFields, together with the field's nested fields.
	// Only the Key, and the NestedFields will be kept as a result, the VirtAccessPoints will be thrown away.
	// Hence it will not show verbose information even if verbose is enabled.
	var flattenNestedFields func(nestedFields StructNestedFields, k StructFieldFullUsageKey, ffu StructFieldFullUsage)
	flattenNestedFields = func(nestedFields StructNestedFields, k StructFieldFullUsageKey, ffu StructFieldFullUsage) {
		nfs, ok := nestedFields[k]
		if !ok {
			nfs = StructFieldFullUsage{
				Key:          k,
				NestedFields: StructNestedFields{},
			}
			nestedFields[k] = nfs
		}

		for k, v := range ffu.NestedFields {
			flattenNestedFields(nfs.NestedFields, k, v)
		}
	}

	for _, fu := range amongAlloc {
		for k, v := range fu.NestedFields {
			flattenNestedFields(out.NestedFields, k, v)
		}
	}
	return out
}

// BuildStructFullUsages extends all the types in rootSet, as long as the type is a structure or interface
// that is implemented by some structures. It only extends the properties (of type structure) when the
// property is directly referenced somewhere, i.e. appears in "dm".
func BuildStructFullUsages(dm StructDirectUsageMap, rootSet NamedTypeAllocSet, opt *StructFullBuildOption) StructFullUsages {
	us := StructFullUsages{
		dm:               dm,
		UsagesAmongAlloc: map[StructFullUsageKey]StructFullUsageAmongAlloc{},
	}

	var wg sync.WaitGroup
	for root, allocSet := range rootSet {
		log.Debugf("building %s\n", root.String())
		us.buildUsagesAmongAlloc(&wg, root, allocSet, opt)
	}
	wg.Wait()
	return us
}
