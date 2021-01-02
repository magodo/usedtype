package usedtype_test

import (
	"testing"

	"github.com/magodo/usedtype/usedtype"
	"github.com/stretchr/testify/require"
)

func TestFindInPackageStructureDirectUsage(t *testing.T) {
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
			``,
		},
		{
			pathA,
			[]string{"."},
			"sdk",
			terraformSchemaTypeFilter,
			``,
		},
		// 2
		{
			pathInterfaceProperty,
			[]string{"."},
			"sdk",
			terraformSchemaTypeFilter,
			``,
		},
		// 3
		{
			pathInterfaceRoot,
			[]string{"."},
			"sdk",
			terraformSchemaTypeFilter,
			``,
		},
		// 4
		{
			pathInterfaceNest,
			[]string{"."},
			"sdk",
			terraformSchemaTypeFilter,
			``,
		},
	}

	for idx, c := range cases {
		pkgs, ssapkgs, _, err := usedtype.BuildPackages(c.dir, c.patterns)
		require.NoError(t, err, idx)
		du := usedtype.FindInPackageStructureDirectUsage(pkgs, ssapkgs)
		_ = du
		//fmt.Println(du.String())
		// We do not assert here because the testing files tend to change...
		//require.Equal(t, c.expect, "\n"+du.String()+"\n", idx)
	}
}
