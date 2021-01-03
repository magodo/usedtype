package usedtype

import (
	"go/types"
	"sort"
	"strings"

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

type VirtAccessNode struct {
	From *VirtAccessNode
	VirtAccessPoint
}

func (node VirtAccessNode) Path() []string {
	depth := 0
	for n := &node; n != nil; n = n.From {
		depth++
	}
	vaps := make([]string, depth)

	for n, i := &node, depth-1; i >= 0; i, n = i-1, n.From {
		vaps[i] = n.VirtAccessPoint.Pos.String()
	}
	return vaps
}

type StructFieldFullUsage struct {
	Key             StructFieldFullUsageKey
	NestedFields    StructNestedFields
	VirtAccessNodes map[VirtAccessNode]struct{}

	dm             StructDirectUsageMap
	seenStructures map[*types.Named]struct{}
}

type StructNestedFields map[StructFieldFullUsageKey]StructFieldFullUsage

type StructFullUsage struct {
	Key          StructFullUsageKey
	NestedFields StructNestedFields

	dm StructDirectUsageMap
}

type StructFullUsages struct {
	dm     StructDirectUsageMap
	Usages map[StructFullUsageKey]StructFullUsage
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
	if len(fu.NestedFields) == 0 {
		return ""
	}
	var out = []string{fu.Key.String()}

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
	var keys StructFullUsageKeys = make([]StructFullUsageKey, len(fus.Usages))
	cnt := 0
	for k := range fus.Usages {
		keys[cnt] = k
		cnt++
	}
	sort.Sort(keys)
	var out []string
	for _, key := range keys {
		if fus.Usages[key].Key.String() == "" {
			continue
		}
		out = append(out, fus.Usages[key].String())
	}
	return strings.Join(out, "\n")
}

func (ffu StructFieldFullUsage) String() string {
	return ffu.stringWithIndent(0)
}

func (ffu StructFieldFullUsage) stringWithIndent(ident int) string {
	prefix := strings.Repeat("  ", ident)
	var out = []string{prefix + ffu.Key.String()}

	if verbose && len(ffu.NestedFields) == 0 {
		for vnode := range ffu.VirtAccessNodes {
			vpath := vnode.Path()
			positions := []string{prefix + "  " + vpath[0]}
			for _, p := range vpath[1:] {
				positions = append(positions, prefix+"  "+"  "+p)
			}
			out = append(out, positions...)
		}
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

	newVNode := make(map[VirtAccessNode]struct{}, len(ffu.VirtAccessNodes))
	for k, v := range ffu.VirtAccessNodes {
		newVNode[k] = v
	}

	return StructFieldFullUsage{
		dm:              ffu.dm,
		Key:             ffu.Key,
		NestedFields:    newNestedFields,
		seenStructures:  newSeenStructs,
		VirtAccessNodes: newVNode,
	}
}

// build build nested fields for a given Named structure or Named interface (baseStruct).
func (nsf StructNestedFields) build(dm StructDirectUsageMap, baseStruct *types.Named, seenStructures map[*types.Named]struct{}, fromNodes map[VirtAccessNode]struct{}, graph *callgraph.Graph) {
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
		nestedFieldIsPrimaryType := !IsElemUnderlyingNamedStructOrInterface(nestedFieldType)

		vAccessNodes := make(map[VirtAccessNode]struct{})

		if fromNodes == nil {
			for _, vap := range vaps {
				vAccessNodes[VirtAccessNode{VirtAccessPoint: vap}] = struct{}{}
			}
		} else {
			// Check whether this virtual access can be tracked back along the access path
		vap_loop:
			for _, vap := range vaps {
				for fromNode := range fromNodes {
					if graph != nil {
						if !checkInstructionReachability(fromNode.Instr, vap.Instr, graph) {
							continue
						}
					}
					vAccessNodes[VirtAccessNode{
						From:            &fromNode,
						VirtAccessPoint: vap,
					}] = struct{}{}

					// In case this nested field is a primary type, as long as we can meet the reachability for one
					// virtual access point, we will just leave the loop and still keeps safety.
					// This is for performance purpose.
					if nestedFieldIsPrimaryType {
						break vap_loop
					}

					// As long as one path could reach, we will continue to evaluate the next virtual access point
					// This is for performance purpose.
					continue vap_loop
				}
			}
		}

		if len(vAccessNodes) == 0 {
			continue
		}

		ffu := StructFieldFullUsage{
			dm:              dm,
			seenStructures:  seenStructures,
			NestedFields:    map[StructFieldFullUsageKey]StructFieldFullUsage{},
			VirtAccessNodes: vAccessNodes,
		}

		if nestedFieldIsPrimaryType {
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
				ffu := ffu.copy()
				if !types.Implements(du, t) {
					continue
				}
				k := StructFieldFullUsageKey{
					StructField: nestedField,
					Variant:     du,
				}
				ffu.Key = k
				ffu.NestedFields.build(dm, du, ffu.seenStructures, ffu.VirtAccessNodes, graph)
				nsf[k] = ffu
			}
		case *types.Struct:
			ffu := ffu.copy()
			k := StructFieldFullUsageKey{
				StructField: nestedField,
			}
			ffu.Key = k
			ffu.NestedFields.build(dm, nt, ffu.seenStructures, ffu.VirtAccessNodes, graph)
			nsf[k] = ffu
		default:
			panic("will never happen")
		}
	}
}

// checkInstructionReachability checks whether two instructions can reach the other in either direction.
// Ideally, for a read field access, we should ensure the read of the parent structure can reach the child field's read;
// Otherwise, for a write field access, we should ensure the write of the child field happens first.
// However, it is non-trivial in SSA to determine whether one instrucution (Field/FieldAddr) is for a later read or write.
// Practically, we ignore this difference here, but simply check whether two instructions can reach the other in either direction.
func checkInstructionReachability(i1, i2 ssa.Instruction, graph *callgraph.Graph) bool {
	if i1.Block() == i2.Block() {
		return true
	}
	if i1.Parent() == i2.Parent() {
		return i1.Block().Dominates(i2.Block()) || i2.Block().Dominates(i1.Block())
	}

	// In case n1 can reach n2, it only means the function enclosing i1 has at least one
	// path that calls the function enclosing i2.
	// TODO: we should ensure the i1 can reach the callsite. We didn't do it here for now
	// since `callgraph.PathSearch` only returns an arbitrary path, whilst we should check
	// all possible paths.
	n1 := graph.Nodes[i1.Parent()]
	n2 := graph.Nodes[i2.Parent()]
	paths1to2 := callgraph.PathSearch(n1, func(n *callgraph.Node) bool {
		return n == n2
	})
	paths2to1 := callgraph.PathSearch(n2, func(n *callgraph.Node) bool {
		return n == n1
	})
	return len(paths1to2) != 0 || len(paths2to1) != 0
}

// buildUsages build usages for one Named type, which is either a structure or an interface. In case of interface, it will
// build for all its implementors.
// The meaning of "build usages" here means to regard the input type as the root structure, recursively iterate its fields to
// check whether the virtual access from this type to this field occurs in the direct usage map. Addtionally, we will ensure
// that the virtual access is reachable back to the place where the root type occurs, in turns of call graph.
func (us StructFullUsages) buildUsages(root *types.Named, graph *callgraph.Graph) {
	// If the target Named type is an interface_property, we shall do the full usage processing
	// on each of its variants that appear in the direct usage map.
	if iRoot, ok := root.Underlying().(*types.Interface); ok {
		for du := range us.dm {
			if !types.Implements(du, iRoot) {
				continue
			}
			k := StructFullUsageKey{
				Named:   root,
				Variant: du,
			}
			us.Usages[k] = StructFullUsage{
				dm:           us.dm,
				Key:          k,
				NestedFields: map[StructFieldFullUsageKey]StructFieldFullUsage{},
			}
			us.Usages[k].NestedFields.build(us.dm, du, map[*types.Named]struct{}{}, nil, graph)
		}
		return
	}

	if _, ok := us.dm[root]; !ok {
		return
	}

	k := StructFullUsageKey{
		Named: root,
	}
	us.Usages[k] = StructFullUsage{
		dm:           us.dm,
		Key:          k,
		NestedFields: map[StructFieldFullUsageKey]StructFieldFullUsage{},
	}
	us.Usages[k].NestedFields.build(us.dm, root, map[*types.Named]struct{}{}, nil, graph)
	return
}

// BuildStructFullUsages extends all the types in rootSet, as long as the type is a structure or interface
// that is implemented by some structures. It will iterate structures' properties, if that property is
// another Named structure or interface, we will try to go on extending the property.
// We only extend the properties (of type structure) when the property is directly referenced somewhere, i.e.,
// appears in "dm".
// If `graph` is non-nil, we will further check the reachability when exteding the properties.
func BuildStructFullUsages(dm StructDirectUsageMap, rootSet NamedTypeSet, graph *callgraph.Graph) StructFullUsages {
	us := StructFullUsages{
		dm:     dm,
		Usages: map[StructFullUsageKey]StructFullUsage{},
	}

	for root := range rootSet {
		log.Debugf("building %s\n", root.String())
		us.buildUsages(root, graph)
	}
	return us
}
