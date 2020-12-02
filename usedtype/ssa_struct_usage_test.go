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
			`sdk.AdditionalInfo
sdk.Metadata
sdk.Properties
    Prop1 (prop1)
    Prop2 (prop2)
sdk.Region
sdk.Req
    Name (name)
    Properties (properties)
        Prop1 (prop1)
        Prop2 (prop2)
sdk.client`,
		},
	}

	for idx, c := range cases {
		pkgs, ssapkgs, err := usedtype.BuildPackages(c.dir, c.patterns)
		require.NoError(t, err, idx)
		directUsage := usedtype.FindInPackageStructureDirectUsage(pkgs, ssapkgs)
		targetRootSet := usedtype.FindExternalPackageStruct(pkgs, c.epattern, nil)
		fus := usedtype.BuildStructFullUsages(directUsage, targetRootSet)
		require.Equal(t, c.expect, fus.String(), idx)
	}
}
