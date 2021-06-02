package usedtype_test

import (
	"regexp"
	"testing"

	"github.com/magodo/usedtype/usedtype"
	"github.com/stretchr/testify/require"
)

func TestFindInPackageFieldUsage(t *testing.T) {
	cases := []struct {
		dir           string
		patterns      []string
		epattern      string
		callGraphType usedtype.CallGraphType
		filter        usedtype.NamedTypeFilter
		expect        string
	}{
		// 0
		{
			pathA,
			[]string{"."},
			"sdk",
			usedtype.CallGraphTypeNA,
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
	}

	for idx, c := range cases {
		pkgs, ssapkgs, graph, err := usedtype.BuildPackages(c.dir, c.patterns, c.callGraphType)
		require.NoError(t, err, idx)
		directUsage := usedtype.FindInPackageStructureDirectUsage(pkgs, ssapkgs)
		targetRootSet := usedtype.FindNamedTypeAllocSetInPackage(pkgs, ssapkgs, regexp.MustCompile(c.epattern), c.filter)
		fus := usedtype.BuildStructFullUsages(directUsage, targetRootSet, graph)
		//fmt.Println(fus.String())
		require.Equal(t, c.expect, "\n"+fus.String()+"\n", idx)
	}
}
