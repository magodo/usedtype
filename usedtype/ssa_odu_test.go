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

func TestUseDefBranches_WalkODUChains(t *testing.T) {
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
			fmt.Sprintf(`
%[1]s:13:2 (new sdk.client (client)): ""
	%[1]s:13:2
%[1]s:13:2 (new sdk.client (client)): ""
	%[1]s:14:23
%[1]s:17:16 (parameter b : bool): ""
	-
%[1]s:20:25 (new sdk.Properties (complit)): ""
	%[1]s:18:6 (phi)
	%[1]s:26:2
%[1]s:20:25 (new sdk.Properties (complit)): "Properties.prop1"
	%[1]s:21:8
	%[1]s:21:8
%[1]s:23:25 (new sdk.Properties (complit)): ""
	%[1]s:18:6 (phi)
	%[1]s:26:2
%[1]s:23:25 (new sdk.Properties (complit)): "Properties.prop2"
	%[1]s:24:8
	%[1]s:24:8
%[1]s:8:2 (local sdk.Req (req)): ""
	%[1]s:14:24
	%[1]s:14:23
%[1]s:8:2 (local sdk.Req (req)): "Req.name"
	-
	%[1]s:9:7
%[1]s:8:2 (local sdk.Req (req)): "Req.properties"
	%[1]s:12:6
	%[1]s:12:6
%[1]s:8:2 (local sdk.Req (req)): "Req.properties.prop1"
	%[1]s:11:6
	%[1]s:11:6
	%[1]s:11:17
	%[1]s:11:17
`, filepath.Join(pathA, "main.go")),
		},
		// 1
		{
			pathValParam,
			[]string{"."},
			"sdk",
			fmt.Sprintf(`
%[1]s:11:2 (new sdk.client (client)): ""
	%[1]s:11:2
%[1]s:11:2 (new sdk.client (client)): ""
	%[1]s:12:23
%[1]s:15:16 (parameter input : string): ""
	%[1]s:16:37
	%[1]s:32:2
%[1]s:19:22 (parameter input : string): ""
	%[1]s:22:3
%[1]s:19:22 (parameter input : string): ""
	%[1]s:28:15
	%[1]s:28:3
%[1]s:19:22 (parameter input : string): ""
	%[1]s:28:20
%[1]s:19:29 (parameter old : string): ""
	%[1]s:24:3
%[1]s:19:34 (parameter new : string): ""
	%[1]s:26:3
%[1]s:31:19 (parameter input : string): ""
	%[1]s:32:2
%[1]s:8:2 (local sdk.Req (req)): ""
	%[1]s:12:24
	%[1]s:12:23
%[1]s:8:2 (local sdk.Req (req)): "Req.name"
	%[1]s:10:6
	%[1]s:10:6
`, filepath.Join(pathValParam, "main.go")),
		},
		// 2
		{
			pathMutateParam,
			[]string{"."},
			"sdk",
			fmt.Sprintf(`
%[1]s:12:2 (new sdk.client (client)): ""
	%[1]s:12:2
%[1]s:12:2 (new sdk.client (client)): ""
	%[1]s:13:23
%[1]s:16:17 (parameter prop : *sdk.Properties): "Properties.prop1"
	%[1]s:17:7
	%[1]s:17:7
%[1]s:8:2 (local sdk.Req (req)): ""
	%[1]s:13:24
	%[1]s:13:23
%[1]s:8:2 (local sdk.Req (req)): "Req.properties"
	-
	%[1]s:9:13
%[1]s:8:2 (local sdk.Req (req)): "Req.properties.prop1"
	%[1]s:11:17
	%[1]s:11:17
	%[1]s:11:12
	%[1]s:17:7
	%[1]s:17:7
%[1]s:9:30 (new sdk.Properties (complit)): ""
	%[1]s:9:13
`, filepath.Join(pathMutateParam, "main.go")),
		},
		// 3
		{
			pathMultiReturn,
			[]string{"."},
			"sdk",
			fmt.Sprintf(`
%[1]s:10:2 (new sdk.client (client)): ""
	%[1]s:10:2
%[1]s:10:2 (new sdk.client (client)): ""
	%[1]s:11:23
%[1]s:15:25 (new sdk.Properties (complit)): ""
	%[1]s:17:2
%[1]s:15:25 (new sdk.Properties (complit)): "Properties.prop1"
	%[1]s:16:7
	%[1]s:16:7
%[1]s:8:2 (local sdk.Req (req)): ""
	%[1]s:11:24
	%[1]s:11:23
%[1]s:8:2 (local sdk.Req (req)): "Req.properties"
	%[1]s:9:6
	%[1]s:9:6
`, filepath.Join(pathMultiReturn, "main.go")),
		},
		// 4
		{
			pathBuildPtrPropInFunctionWithIf,
			[]string{"."},
			"sdk",
			fmt.Sprintf(`
%[1]s:15:16 (parameter b : bool): ""
	-
	not used anywhere
%[1]s:18:25 (new sdk.Properties (complit)): ""
	%[1]s:16:6 (phi)
	%[1]s:24:2
	%[1]s:11:6
	waiting for consumer
%[1]s:18:25 (new sdk.Properties (complit)): "Properties.prop1"
	%[1]s:19:8
	%[1]s:19:8
	waiting for provider
%[1]s:21:25 (new sdk.Properties (complit)): ""
	%[1]s:16:6 (phi)
	%[1]s:24:2
	%[1]s:11:6
	waiting for consumer
%[1]s:21:25 (new sdk.Properties (complit)): "Properties.prop2"
	%[1]s:22:8
	%[1]s:22:8
	waiting for provider
%[1]s:8:2 (local sdk.Req (req)): ""
	%[1]s:12:6
	not used anywhere
%[1]s:8:2 (local sdk.Req (req)): "Req.name"
	-
	%[1]s:9:7
	waiting for provider
%[1]s:8:2 (local sdk.Req (req)): "Req.properties"
	%[1]s:11:6
	%[1]s:11:6
	waiting for provider
`, filepath.Join(pathBuildPtrPropInFunctionWithIf, "main.go")),
		},
	}

	for idx, c := range cases {
		if idx != 4 {
			continue
		}
		pkgs, ssapkgs, err := usedtype.BuildPackages(c.dir, c.patterns)
		require.NoError(t, err, idx)
		ssadefs := usedtype.FindInPackageAllDefValue(pkgs, ssapkgs)
		var chains []string
		for _, value := range ssadefs {
			oduChains := usedtype.WalkODUChains(value.Value, ssapkgs, value.Fset)
			for _, chain := range oduChains {
				chains = append(chains, chain.String())
			}
		}
		sort.Strings(chains)
		fmt.Println(strings.Join(chains, "\n"))
		require.Equal(t, c.expect, "\n"+strings.Join(chains, "\n")+"\n", idx)
	}
}

func TestUseDefBranches_Pair(t *testing.T) {
	cases := []struct {
		dir      string
		patterns []string
		epattern string
		expect   string
	}{
		// 0
		//{
		//	pathBuildPtrPropInFunctionWithIf,
		//	[]string{"."},
		//	"sdk",
		//	"",
		//},
		// 1
		{
			pathBuildNestedPropInFunction,
			[]string{"."},
			"sdk",
			"",
		},
	}

	for idx, c := range cases {
		pkgs, ssapkgs, err := usedtype.BuildPackages(c.dir, c.patterns)
		require.NoError(t, err, idx)
		ssadefs := usedtype.FindInPackageAllDefValue(pkgs, ssapkgs)
		var allOduChains usedtype.ODUChains
		for _, value := range ssadefs {
			oduChains := usedtype.WalkODUChains(value.Value, ssapkgs, value.Fset)
			for _, chain := range oduChains {
				allOduChains = append(allOduChains, chain)
			}
		}
		var tmpchains []string
		for _, c := range allOduChains {
			tmpchains = append(tmpchains, c.String())
		}
		sort.Strings(tmpchains)
		fmt.Println(strings.Join(tmpchains, "\n"))
		fmt.Println("###########################")

		oduChains := allOduChains.Pair()
		var chains []string
		for _, c := range oduChains {
			chains = append(chains, c.String())
		}
		sort.Strings(chains)
		fmt.Println(strings.Join(chains, "\n"))
		//require.Equal(t, c.expect, "\n"+strings.Join(chains, "\n")+"\n", idx)
	}
}
