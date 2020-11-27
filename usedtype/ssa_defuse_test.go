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
	%[1]s:17:6
	%[1]s:18:6
	%[1]s:20:25
"Req.properties"
	%[1]s:8:2
	%[1]s:12:6
	%[1]s:12:28
	%[1]s:17:6
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
	%[1]s:17:6
	%[1]s:18:6
	%[1]s:20:25
	%[1]s:21:8
	-
"Req.properties.prop2"
	%[1]s:8:2
	%[1]s:12:6
	%[1]s:12:28
	%[1]s:17:6
	%[1]s:18:6
	%[1]s:23:25
	%[1]s:24:8
	-
`, filepath.Join(pathA, "main.go")),
			},
		},
		// 1
		{
			pathValParam,
			[]string{"."},
			"sdk",
			terraformSchemaTypeFilter,
			map[string]string{
				"Req (sdk)": fmt.Sprintf(`""
	%[1]s:8:2
	%[1]s:12:24
"Req.name"
	%[1]s:8:2
	%[1]s:10:6
	%[1]s:9:19
	%[1]s:15:6
	%[1]s:16:24
	%[1]s:19:6
	%[1]s:19:22
"Req.name"
	%[1]s:8:2
	%[1]s:10:6
	%[1]s:9:19
	%[1]s:15:6
	%[1]s:16:24
	%[1]s:19:6
	%[1]s:28:15
`, filepath.Join(pathValParam, "main.go")),
			},
		},
		// 2
		{
			pathMutateParam,
			[]string{"."},
			"sdk",
			terraformSchemaTypeFilter,
			map[string]string{
				"Req (sdk)": fmt.Sprintf(`""
	%[1]s:8:2
	%[1]s:13:24
"Req.properties"
	%[1]s:8:2
	-
	%[1]s:9:30
"Req.properties.prop1"
	%[1]s:8:2
	%[1]s:11:17
	%[1]s:11:17
	%[1]s:11:12
	%[1]s:16:6
	%[1]s:16:17
	%[1]s:17:7
	-
`, filepath.Join(pathMutateParam, "main.go")),
			},
		},
		// 3
		{
			pathMultiReturn,
			[]string{"."},
			"sdk",
			terraformSchemaTypeFilter,
			map[string]string{
				"Req (sdk)": fmt.Sprintf(`""
	%[1]s:8:2
	%[1]s:11:24
"Req.properties"
	%[1]s:8:2
	%[1]s:9:6
	-
	%[1]s:9:31
	%[1]s:14:6
	%[1]s:15:25
"Req.properties.prop1"
	%[1]s:8:2
	%[1]s:9:6
	-
	%[1]s:9:31
	%[1]s:14:6
	%[1]s:15:25
	%[1]s:16:7
	-
`, filepath.Join(pathMultiReturn, "main.go")),
			},
		},
	}

	for idx, c := range cases {
		if idx != 0 {
			continue
		}
		pkgs, ssapkgs, err := usedtype.BuildPackages(c.dir, c.patterns)
		ssadefs := usedtype.FindInPackageAllDefValue(pkgs, ssapkgs)
		var chains []string
		for _, value := range ssadefs {
			oduChains := usedtype.WalkODUChains(value.Value, ssapkgs, value.Fset)
			for _, chain := range oduChains {
				chains = append(chains, chain.String())
			}
		}
		require.NoError(t, err, idx)
		sort.Strings(chains)
		fmt.Println("================")
		fmt.Println(strings.Join(chains, "\n"))
		//require.Equal(t, c.expect[tid.String()], strings.Join(chains, ""), idx)
	}
}
