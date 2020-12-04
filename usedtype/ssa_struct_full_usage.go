package usedtype

import (
	"go/types"
	"sort"
	"strings"
)

type StructFullUsageKey struct {
	named   *types.Named
	variant *types.Named // non-nil only when named is a named interface_property
}

type StructFullUsageKeys []StructFullUsageKey

type StructFieldFullUsageKey struct {
	structField
	variant *types.Named // non-nil only when the structField corresponds to an interface_property
}

type StructFieldFullUsageKeys []StructFieldFullUsageKey

func (key StructFullUsageKey) String() string {
	if key.variant == nil {
		return key.named.String()
	}
	return key.named.String() + " [" + key.variant.String() + "]"
}

func (keys StructFullUsageKeys) Len() int {
	return len(keys)
}
func (keys StructFullUsageKeys) Swap(i, j int) {
	keys[i], keys[j] = keys[j], keys[i]
}

func (keys StructFullUsageKeys) Less(i, j int) bool {
	return keys[i].named.String() < keys[j].named.String() ||
		(keys[i].named.String() == keys[j].named.String() &&
			keys[i].variant.String() < keys[j].variant.String())
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
			keys[i].variant.String() < keys[j].variant.String())
}

func (key StructFieldFullUsageKey) String() string {
	if key.variant == nil {
		return key.structField.String()
	}
	return key.structField.String() + " [" + key.variant.String() + "]"
}

type StructFieldFullUsage struct {
	dm             StructDirectUsageMap
	key            StructFieldFullUsageKey
	seenStructures map[*types.Named]struct{}
	nestedFields   StructNestedFields
}

type StructNestedFields map[StructFieldFullUsageKey]StructFieldFullUsage

type StructFullUsage struct {
	dm           StructDirectUsageMap
	key          StructFullUsageKey
	nestedFields StructNestedFields
}

type StructFullUsages struct {
	dm     StructDirectUsageMap
	usages map[StructFullUsageKey]StructFullUsage
}

func (fu StructFullUsage) String() string {
	if len(fu.nestedFields) == 0 {
		return ""
	}
	var out = []string{fu.key.String()}

	var keys StructFieldFullUsageKeys = make([]StructFieldFullUsageKey, len(fu.nestedFields))
	cnt := 0
	for k := range fu.nestedFields {
		keys[cnt] = k
		cnt++
	}
	sort.Sort(keys)

	for _, key := range keys {
		out = append(out, fu.nestedFields[key].StringWithIndent(2))
	}
	return strings.Join(out, "\n")
}

func (fus StructFullUsages) String() string {
	var keys StructFullUsageKeys = make([]StructFullUsageKey, len(fus.usages))
	cnt := 0
	for k := range fus.usages {
		keys[cnt] = k
		cnt++
	}
	sort.Sort(keys)
	var out []string
	for _, key := range keys {
		if fus.usages[key].key.String() == "" {
			continue
		}
		out = append(out, fus.usages[key].String())
	}
	return strings.Join(out, "\n")
}

func (ffu StructFieldFullUsage) String() string {
	return ffu.StringWithIndent(0)
}

func (ffu StructFieldFullUsage) StringWithIndent(ident int) string {
	prefix := strings.Repeat("  ", ident)
	var out = []string{prefix + ffu.key.String()}

	var keys StructFieldFullUsageKeys = make([]StructFieldFullUsageKey, len(ffu.nestedFields))
	cnt := 0
	for k := range ffu.nestedFields {
		keys[cnt] = k
		cnt++
	}
	sort.Sort(keys)

	for _, key := range keys {
		out = append(out, ffu.nestedFields[key].StringWithIndent(ident+2))
	}
	return strings.Join(out, "\n")
}

func (ffu StructFieldFullUsage) copy() StructFieldFullUsage {
	newNestedFields := make(map[StructFieldFullUsageKey]StructFieldFullUsage)
	for k, v := range ffu.nestedFields {
		newNestedFields[k] = v.copy()
	}

	newSeenStructs := make(map[*types.Named]struct{})
	for k, v := range ffu.seenStructures {
		newSeenStructs[k] = v
	}

	return StructFieldFullUsage{
		dm:             ffu.dm,
		key:            ffu.key,
		nestedFields:   newNestedFields,
		seenStructures: newSeenStructs,
	}
}

func (nsf StructNestedFields) build(dm StructDirectUsageMap, baseStruct *types.Named, seenStructures map[*types.Named]struct{}) {
	if _, ok := seenStructures[baseStruct]; ok {
		return
	}
	seenStructures[baseStruct] = struct{}{}

	du, ok := dm[baseStruct]
	if !ok {
		return
	}

	for nestedField := range du {
		ffu := StructFieldFullUsage{
			dm:             dm,
			seenStructures: seenStructures,
			nestedFields:   map[StructFieldFullUsageKey]StructFieldFullUsage{},
		}

		t := nestedField.DereferenceRElem()
		if !IsElemUnderlyingNamedStructOrInterface(t) {
			k := StructFieldFullUsageKey{
				structField: nestedField,
			}
			ffu.key = k
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
					structField: nestedField,
					variant:     du,
				}
				ffu.key = k
				ffu.nestedFields.build(dm, du, ffu.seenStructures)
				nsf[k] = ffu
			}
		case *types.Struct:
			ffu := ffu.copy()
			k := StructFieldFullUsageKey{
				structField: nestedField,
			}
			ffu.key = k
			ffu.nestedFields.build(dm, nt, ffu.seenStructures)
			nsf[k] = ffu
		default:
			panic("will never happen")
		}
	}
}

func (us StructFullUsages) buildUsages(root *types.Named) {
	// If the target named type is an interface_property, we shall do the full usage processing
	// on each of its variants that appear in the direct usage map.
	if iRoot, ok := root.Underlying().(*types.Interface); ok {
		for du := range us.dm {
			if !types.Implements(du, iRoot) {
				continue
			}
			k := StructFullUsageKey{
				named:   root,
				variant: du,
			}
			us.usages[k] = StructFullUsage{
				dm:           us.dm,
				key:          k,
				nestedFields: map[StructFieldFullUsageKey]StructFieldFullUsage{},
			}
			us.usages[k].nestedFields.build(us.dm, du, map[*types.Named]struct{}{})
		}
		return
	}

	if _, ok := us.dm[root]; !ok {
		return
	}

	k := StructFullUsageKey{
		named: root,
	}
	us.usages[k] = StructFullUsage{
		dm:           us.dm,
		key:          k,
		nestedFields: map[StructFieldFullUsageKey]StructFieldFullUsage{},
	}
	us.usages[k].nestedFields.build(us.dm, root, map[*types.Named]struct{}{})
	return
}

func BuildStructFullUsages(dm StructDirectUsageMap, rootSet NamedTypeSet) StructFullUsages {
	us := StructFullUsages{
		dm:     dm,
		usages: map[StructFullUsageKey]StructFullUsage{},
	}

	for root := range rootSet {
		us.buildUsages(root)
	}
	return us
}
