package usedtype_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/magodo/usedtype/usedtype"
	"github.com/stretchr/testify/require"
)

func TestFindInPackageDefNodeOfTargetStructType(t *testing.T) {
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
			fmt.Sprintf(`Properties (sdk)
	%[1]s:20:25
	%[1]s:23:25
Req (sdk)
	%[1]s:8:2
client (sdk)
	%[1]s:13:2
`, filepath.Join(pathA, "main.go")),
		},
		// 1
		{
			pathA,
			[]string{"."},
			"sdk",
			terraformSchemaTypeFilter,
			fmt.Sprintf(`Req (sdk)
	%[1]s:8:2
`, filepath.Join(pathA, "main.go")),
		},
	}

	for idx, c := range cases {
		pkgs, ssapkgs, err := usedtype.BuildPackages(c.dir, c.patterns)
		require.NoError(t, err, idx)
		structNodes := usedtype.FindInPackageDefNodeOfTargetStructType(ssapkgs, usedtype.FindExternalPackageStruct(pkgs, c.epattern, c.filter))
		require.Equal(t, c.expect, structNodes.String(), idx)

	}
}
