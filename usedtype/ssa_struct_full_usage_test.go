package usedtype_test

import (
	"fmt"
	"go/types"
	"regexp"
	"testing"

	"github.com/magodo/usedtype/usedtype"
	"github.com/stretchr/testify/require"
)

func azureSDKTrack1Implements(v types.Type, nt *types.Named) bool {
	t, ok := nt.Underlying().(*types.Interface)
	if !ok {
		return false
	}
	// Store the struct types that implement this interface.
	implementors := []*types.Named{}

	// Store the interfaces that inherit this interface, including itself.
	interfaces := map[string]bool{
		nt.Obj().Name(): true,
	}

	for i := 0; i < t.NumMethods(); i++ {
		signature, ok := t.Method(i).Type().(*types.Signature)
		if !ok {
			continue
		}

		methodReturns := signature.Results()

		// The return type is always (struct ptr/interface, bool)
		if methodReturns.Len() != 2 {
			continue
		}

		vt := methodReturns.At(0).Type()
		vt = usedtype.DereferenceR(vt)
		nt, ok := vt.(*types.Named)
		if !ok {
			continue
		}

		ut := nt.Underlying()

		switch ut.(type) {
		case *types.Interface:
			interfaces[nt.Obj().Name()] = true
		case *types.Struct:
			implementors = append(implementors, nt)
		}
	}

	for _, nt := range implementors {
		// Skip the hypothetic base types from the implementers
		if interfaces["Basic"+nt.Obj().Name()] {
			continue
		}
		if types.Identical(nt, v) {
			return true
		}
	}

	return false
}

func TestFindInPackageFieldUsage(t *testing.T) {
	cases := []struct {
		dir           string
		patterns      []string
		epattern      string
		callGraphType usedtype.CallGraphType
		filter        usedtype.NamedTypeFilter
		impl          usedtype.CustomImplements
		expect        string
	}{
		// 0
		{
			pathA,
			[]string{"."},
			"sdk",
			usedtype.CallGraphTypeNA,
			nil,
			nil,
			`
sdk.ModelA
    String (string)
    Property (property)
        Int (int)
    PointerOfProperty (pointer_of_property)
        Int (int)
    ArrayOfString (array_of_string)
    PointerOfArrayOfString (pointer_of_array_of_string)
    ArrayOfProperty (array_of_property)
        Int (int)
    PointerOfArrayOfProperty (pointer_of_array_of_property)
        Int (int)
    ArrayOfPointerOfProperty (array_of_pointer_of_property)
        Int (int)
sdk.Property
    Int (int)
`,
		},
		// 1
		{
			pathInterfaceProperty,
			[]string{"."},
			"sdk",
			usedtype.CallGraphTypeNA,
			filterTypeByName("sdk.AnimalFamily"),
			nil,
			`
sdk.AnimalFamily [sdk.DogFamily]
    Animals (animals) [sdk.Dog]
        Name (name)
        RunSpeed (run_speed)
    Animals (animals) [sdk.Fish]
        Name (name)
        SwimSpeed (swim_speed)
`,
		},
		// 2
		{
			pathInterfaceRoot,
			[]string{"."},
			"sdk",
			usedtype.CallGraphTypeNA,
			filterTypeByName("sdk.Animal"),
			nil,
			`
sdk.Animal [sdk.Dog]
    Name (name)
    RunSpeed (run_speed)
sdk.Animal [sdk.Fish]
    Name (name)
    SwimSpeed (swim_speed)
`,
		},
		// 3
		{
			pathInterfaceNest,
			[]string{"."},
			"sdk",
			usedtype.CallGraphTypeNA,
			filterTypeByName("sdk.Zoo"),
			nil,
			`
sdk.Zoo
    AnimalFamilies (animal_family) [sdk.BirdFamily]
        Animals (animals) [sdk.Bird]
            Name (name)
        Animals (animals) [sdk.Dog]
            Name (name)
        Animals (animals) [sdk.Fish]
            Name (name)
    AnimalFamilies (animal_family) [sdk.DogFamily]
        Animals (animals) [sdk.Bird]
            Name (name)
        Animals (animals) [sdk.Dog]
            Name (name)
        Animals (animals) [sdk.Fish]
            Name (name)
    AnimalFamilies (animal_family) [sdk.FishFamily]
        Animals (animals) [sdk.Bird]
            Name (name)
        Animals (animals) [sdk.Dog]
            Name (name)
        Animals (animals) [sdk.Fish]
            Name (name)
`,
		},
		// 4
		{
			pathCrossFunc,
			[]string{"."},
			"sdk",
			usedtype.CallGraphTypeNA,
			filterTypeByName("sdk.ModelA"),
			nil,
			`
sdk.ModelA
    String (string)
    Property (property)
        Int (int)
`,
		},
		// 5
		{
			pathCrossFunc,
			[]string{"."},
			"sdk",
			usedtype.CallGraphTypeStatic,
			filterTypeByName("sdk.ModelA"),
			nil,
			`
sdk.ModelA
    String (string)
    Property (property)
`,
		},
		// 6
		{
			pathCrossBB,
			[]string{"."},
			"sdk",
			usedtype.CallGraphTypeStatic,
			filterTypeByName("sdk.ModelA"),
			nil,
			`
sdk.ModelA
    ArrayOfProperty (array_of_property)
        Int (int)
`,
		},
		// 7
		{
			pathCrossFuncNoLink,
			[]string{"."},
			"sdk",
			usedtype.CallGraphTypeStatic,
			filterTypeByName("sdk.ModelA"),
			nil,
			`
sdk.ModelA
    String (string)
    Property (property)
        Int (int)
    ArrOfPropWrapper (array_of_prop_wrapper)
        Prop (prop)
            Int (int)
`,
		},
		// 8
		{
			pathInterfaceNestAzureSDKTrack1,
			[]string{"."},
			"sdk",
			usedtype.CallGraphTypeNA,
			filterTypeByName("sdk.BasicMiddle"),
			azureSDKTrack1Implements,
			`
sdk.BasicMiddle [sdk.B]
    Name
sdk.BasicMiddle [sdk.C]
    Name
`,
		},
		// 8
		{
			pathInitMethod,
			[]string{"."},
			"foo",
			usedtype.CallGraphTypeNA,
			nil,
			nil,
			`
a/foo.Foo
    Bar
        Name
`,
		},
	}

	for idx, c := range cases {
		pkgs, ssapkgs, graph, err := usedtype.BuildPackages(c.dir, c.patterns, c.callGraphType)
		require.NoError(t, err, idx)
		directUsage := usedtype.FindInPackageStructureDirectUsage(pkgs, ssapkgs)
		targetRootSet := usedtype.FindNamedTypeAllocSetInPackage(pkgs, ssapkgs, regexp.MustCompile(c.epattern), c.filter)
		fus := usedtype.BuildStructFullUsages(directUsage, targetRootSet,
			&usedtype.StructFullBuildOption{
				Callgraph:        graph,
				CustomImplements: c.impl,
			},
		)
		fmt.Println(fus.String())
		require.Equal(t, c.expect, "\n"+fus.String()+"\n", idx)
	}
}
