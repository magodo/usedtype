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
		expect   string
	}{
		// 0
		{
			pathA,
			[]string{"."},
			"sdk",
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
			pathInterface,
			[]string{"."},
			"sdk",
			`
`,
		},
	}

	for idx, c := range cases {
		if idx != 1 {
			continue
		}
		pkgs, ssapkgs, err := usedtype.BuildPackages(c.dir, c.patterns)
		require.NoError(t, err, idx)
		directUsage := usedtype.FindInPackageStructureDirectUsage(pkgs, ssapkgs)
		targetRootSet := usedtype.FindExternalPackageStruct(pkgs, c.epattern, nil)
		fus := usedtype.BuildStructFullUsages(directUsage, targetRootSet)
		//fmt.Println(fus.String())
		require.Equal(t, c.expect, "\n"+fus.String()+"\n", idx)
	}
}
