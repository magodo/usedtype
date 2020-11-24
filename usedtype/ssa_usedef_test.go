package usedtype_test

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/magodo/usedtype/usedtype"
	"github.com/stretchr/testify/require"
)

func TestUseDefBranches_Walk(t *testing.T) {
	cases := []struct {
		dir      string
		patterns []string
		epattern string
		filter   usedtype.FilterFunc
		expect   map[string]string
	}{
		// 0
		{
			pathA,
			[]string{"."},
			"sdk",
			terraformSchemaTypeFilter,
			map[string]string{
				"Req (sdk)": fmt.Sprintf(`""
	%[1]s:8:2
	%[1]s:14:24
"Req.name"
	%[1]s:8:2
	-
	-
"Req.properties"
	%[1]s:8:2
	%[1]s:12:6
	%[1]s:12:28
	%[1]s:18:6
	%[1]s:20:25
"Req.properties"
	%[1]s:8:2
	%[1]s:12:6
	%[1]s:12:28
	%[1]s:18:6
	%[1]s:23:25
"Req.properties.prop1"
	%[1]s:8:2
	%[1]s:11:6
	%[1]s:11:6
	%[1]s:11:17
	-
"Req.properties.prop1"
	%[1]s:8:2
	%[1]s:12:6
	%[1]s:12:28
	%[1]s:18:6
	%[1]s:20:25
	%[1]s:21:8
	-
"Req.properties.prop2"
	%[1]s:8:2
	%[1]s:12:6
	%[1]s:12:28
	%[1]s:18:6
	%[1]s:23:25
	%[1]s:24:8
	-
`, filepath.Join(pathA, "main.go")),
			},
		},
	}

	for idx, c := range cases {
		pkgs, ssapkgs, err := usedtype.BuildPackages(c.dir, c.patterns)
		require.NoError(t, err, idx)
		structs := usedtype.FindExternalPackageStruct(pkgs, c.epattern, c.filter)
		structnodes := usedtype.FindInPackageDefNodeOfTargetStructType(ssapkgs, structs)
		for tid, values := range structnodes {
			chains := []string{}
			for _, value := range values {
				var branches usedtype.DefUseBranches
				branches = usedtype.NewDefUseBranches(value, tid.Pkg.Fset)
				newbranches := branches.Walk()
				for _, b := range newbranches {
					chains = append(chains, b.String())
				}
			}
			sort.Strings(chains)
			require.Equal(t, c.expect[tid.String()], strings.Join(chains, ""), idx)
		}
	}
}
