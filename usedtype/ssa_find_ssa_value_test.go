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
		structNodes := usedtype.FindInPackageDefValueOfTargetStructType(ssapkgs, usedtype.FindExternalPackageStruct(pkgs, c.epattern, c.filter))
		require.Equal(t, c.expect, structNodes.String(), idx)

	}
}

func TestFindInPackageAllDefNode(t *testing.T) {
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
			fmt.Sprintf(`%[1]s:13:2: new sdk.client (client)
%[1]s:17:16: parameter b : bool
%[1]s:20:25: new sdk.Properties (complit)
%[1]s:23:25: new sdk.Properties (complit)
%[1]s:8:2: local sdk.Req (req)
`, filepath.Join(pathA, "main.go")),
		},
		// 1
		{
			pathValParam,
			[]string{"."},
			"sdk",
			fmt.Sprintf(`%[1]s:11:2: new sdk.client (client)
%[1]s:15:16: parameter input : string
%[1]s:19:22: parameter input : string
%[1]s:19:29: parameter old : string
%[1]s:19:34: parameter new : string
%[1]s:31:19: parameter input : string
%[1]s:8:2: local sdk.Req (req)
`, filepath.Join(pathValParam, "main.go")),
		},
		// 2
		{
			pathMultiReturn,
			[]string{"."},
			"sdk",
			fmt.Sprintf(`%[1]s:10:2: new sdk.client (client)
%[1]s:15:25: new sdk.Properties (complit)
%[1]s:8:2: local sdk.Req (req)
`, filepath.Join(pathMultiReturn, "main.go")),
		},
		// 3
		{
			pathMutateParam,
			[]string{"."},
			"sdk",
			fmt.Sprintf(`%[1]s:12:2: new sdk.client (client)
%[1]s:16:17: parameter prop : *sdk.Properties
%[1]s:8:2: local sdk.Req (req)
%[1]s:9:30: new sdk.Properties (complit)
`, filepath.Join(pathMutateParam, "main.go")),
		},
	}

	for idx, c := range cases {
		pkgs, ssapkgs, err := usedtype.BuildPackages(c.dir, c.patterns)
		require.NoError(t, err, idx)
		ssavalues := usedtype.FindInPackageAllDefValue(pkgs, ssapkgs)
		require.Equal(t, c.expect, ssavalues.String(), idx)
	}
}
