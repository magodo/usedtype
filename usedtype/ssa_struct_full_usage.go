package usedtype

import (
	"go/token"
	"go/types"
	"sort"
	"strings"
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
	Key          StructFieldFullUsageKey
	NestedFields StructNestedFields
	Positions    []token.Position

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

	if verbose {
		positions := []string{}
		for _, pos := range ffu.Positions {
			positions = append(positions, prefix+pos.String())
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

	newPositions := make([]token.Position, len(ffu.Positions))
	copy(newPositions, ffu.Positions)

	return StructFieldFullUsage{
		dm:             ffu.dm,
		Key:            ffu.Key,
		NestedFields:   newNestedFields,
		seenStructures: newSeenStructs,
		Positions:      newPositions,
	}
}

// build build nested fields for a given Named structure or Named interface (baseStruct).
func (nsf StructNestedFields) build(dm StructDirectUsageMap, baseStruct *types.Named, seenStructures map[*types.Named]struct{}) {
	if _, ok := seenStructures[baseStruct]; ok {
		return
	}
	seenStructures[baseStruct] = struct{}{}

	du, ok := dm[baseStruct]
	if !ok {
		return
	}

	for nestedField, positions := range du {
		ffu := StructFieldFullUsage{
			dm:             dm,
			seenStructures: seenStructures,
			NestedFields:   map[StructFieldFullUsageKey]StructFieldFullUsage{},
			Positions:      positions,
		}

		t := nestedField.DereferenceRElem()
		if !IsElemUnderlyingNamedStructOrInterface(t) {
			k := StructFieldFullUsageKey{
				StructField: nestedField,
			}
			ffu.Key = k
			nsf[k] = ffu
			continue
		}

		nt := t.(*types.Named)
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
				ffu.NestedFields.build(dm, du, ffu.seenStructures)
				nsf[k] = ffu
			}
		case *types.Struct:
			ffu := ffu.copy()
			k := StructFieldFullUsageKey{
				StructField: nestedField,
			}
			ffu.Key = k
			ffu.NestedFields.build(dm, nt, ffu.seenStructures)
			nsf[k] = ffu
		default:
			panic("will never happen")
		}
	}
}

// buildUsages build Usages for one Named type. When it is a structure, it will build usage for the structure as long
// as the structure appears in the "dm". When it is an interface, it will build usage for all structures appear in the "dm"
// that implement the interface.
func (us StructFullUsages) buildUsages(root *types.Named) {
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
			us.Usages[k].NestedFields.build(us.dm, du, map[*types.Named]struct{}{})
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
	us.Usages[k].NestedFields.build(us.dm, root, map[*types.Named]struct{}{})
	return
}

// BuildStructFullUsages extends all the types in rootSet, as long as the type is a structure or interface
// that is implemented by some structures. It will iterate structures' properties, if that property is
// another Named structure or interface, we will try to go on extending the property.
// We only extend the properties (of type structure) when the property is directly referenced somewhere, i.e.,
// appears in "dm".
func BuildStructFullUsages(dm StructDirectUsageMap, rootSet NamedTypeSet) StructFullUsages {
	us := StructFullUsages{
		dm:     dm,
		Usages: map[StructFullUsageKey]StructFullUsage{},
	}

	for root := range rootSet {
		us.buildUsages(root)
	}
	return us
}
