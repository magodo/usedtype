package usedtype_test

import (
	"path/filepath"
	"testing"

	"github.com/magodo/usedtype/usedtype"
	"github.com/stretchr/testify/require"
)

func TestFindExternalPackageStruct(t *testing.T) {
	cases := []struct {
		dir      string
		patterns []string
		epattern string
		filter   usedtype.FilterFunc
		expect   string
	}{
		// 0
		{
			filepath.Join("testdata", "src", "a"),
			[]string{"."},
			"sdk",
			nil,
			`Properties (sdk): struct{Prop1 int "json:\"prop1\""; Prop2 string "json:\"prop2\""}
Req (sdk): struct{Name string "json:\"name,omitempty\""; *sdk.Properties "json:\"properties,omitempty\""}
client (sdk): struct{}
`,
		},
		// 1
		{
			filepath.Join("testdata", "src", "a"),
			[]string{"."},
			"sdk",
			terraformSchemaTypeFilter,
			`Req (sdk): struct{Name string "json:\"name,omitempty\""; *sdk.Properties "json:\"properties,omitempty\""}
`,
		},
	}

	for idx, c := range cases {
		pkgs, _, err := usedtype.BuildPackages(c.dir, c.patterns)
		require.NoError(t, err, idx)
		structs := usedtype.FindExternalPackageStruct(pkgs, c.epattern, c.filter)
		require.Equal(t, c.expect, structs.String(), idx)

	}
}
