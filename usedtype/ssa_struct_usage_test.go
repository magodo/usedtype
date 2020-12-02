package usedtype_test

import (
	"fmt"
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
		//// 0
		//{
		//	pathBuildPtrPropInFunctionWithIf,
		//	[]string{"."},
		//	"sdk",
		//	"",
		//},
		//// 1
		//{
		//	pathBuildNestedPropInFunction,
		//	[]string{"."},
		//	"sdk",
		//	"",
		//},
		// 2
		{
			pathArrayProp,
			[]string{"."},
			"sdk",
			"",
		},
		//// 3
		//{
		//	pathInvoke,
		//	[]string{"."},
		//	"sdk",
		//	"",
		//},
	}

	for idx, c := range cases {
		pkgs, ssapkgs, err := usedtype.BuildPackages(c.dir, c.patterns)
		require.NoError(t, err, idx)
		usages := usedtype.FindInPackageStructureDirectUsage(pkgs, ssapkgs)
		fmt.Println(usages)
	}
}
