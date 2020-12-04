package usedtype_test

import (
	"testing"

	"github.com/magodo/usedtype/usedtype"
	"github.com/stretchr/testify/require"
)

func TestFindInPackageFieldUsage(t *testing.T) {
	cases := []struct {
		dir      string
		patterns []string
		epattern string
		filter   usedtype.FilterFunc
		expect   string
	}{
		// 0
		{
			pathA,
			[]string{"."},
			"sdk",
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
		{
			pathA,
			[]string{"."},
			"sdk",
			terraformSchemaTypeFilter,
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
`,
		},
		// 2
		{
			pathInterfaceProperty,
			[]string{"."},
			"sdk",
			terraformSchemaTypeFilter,
			`
sdk.Animal [sdk.Dog]
    Name (name)
    RunSpeed (run_speed)
sdk.Animal [sdk.Fish]
    Name (name)
    SwimSpeed (swim_speed)
sdk.AnimalFamily [sdk.DogFamily]
    Animals (animals) [sdk.Dog]
        Name (name)
        RunSpeed (run_speed)
    Animals (animals) [sdk.Fish]
        Name (name)
        SwimSpeed (swim_speed)
`,
		},
		// 3
		{
			pathInterfaceRoot,
			[]string{"."},
			"sdk",
			terraformSchemaTypeFilter,
			`
sdk.Animal [sdk.Dog]
    Name (name)
    RunSpeed (run_speed)
sdk.Animal [sdk.Fish]
    Name (name)
    SwimSpeed (swim_speed)
`,
		},
		// 4
		{
			pathInterfaceNest,
			[]string{"."},
			"sdk",
			terraformSchemaTypeFilter,
			`
sdk.Animal [sdk.Bird]
    Name (name)
sdk.Animal [sdk.Dog]
    Name (name)
sdk.Animal [sdk.Fish]
    Name (name)
sdk.AnimalFamily [sdk.BirdFamily]
    Animals (animals) [sdk.Bird]
        Name (name)
    Animals (animals) [sdk.Dog]
        Name (name)
    Animals (animals) [sdk.Fish]
        Name (name)
sdk.AnimalFamily [sdk.DogFamily]
    Animals (animals) [sdk.Bird]
        Name (name)
    Animals (animals) [sdk.Dog]
        Name (name)
    Animals (animals) [sdk.Fish]
        Name (name)
sdk.AnimalFamily [sdk.FishFamily]
    Animals (animals) [sdk.Bird]
        Name (name)
    Animals (animals) [sdk.Dog]
        Name (name)
    Animals (animals) [sdk.Fish]
        Name (name)
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
	}

	for idx, c := range cases {
		pkgs, ssapkgs, err := usedtype.BuildPackages(c.dir, c.patterns)
		require.NoError(t, err, idx)
		directUsage := usedtype.FindInPackageStructureDirectUsage(pkgs, ssapkgs)
		targetRootSet := usedtype.FindExternalPackageNamedType(pkgs, c.epattern, c.filter)
		fus := usedtype.BuildStructFullUsages(directUsage, targetRootSet)
		//fmt.Println(fus.String())
		require.Equal(t, c.expect, "\n"+fus.String()+"\n", idx)
	}
}
