package usedtype_test

import (
	"fmt"
	"testing"

	"golang.org/x/tools/go/ssa"

	"github.com/magodo/usedtype/usedtype"
	"github.com/stretchr/testify/require"
)

func TestInstrPos(t *testing.T) {
	pkgs, ssapkgs, _, err := usedtype.BuildPackages(pathInstrPos, []string{"."}, usedtype.CallGraphTypeNA)
	require.NoError(t, err)
	_ = ssapkgs

	instrs := ssapkgs[0].Members["main"].(*ssa.Function).Blocks[0].Instrs
	fset := pkgs[0].Fset

	cases := []struct {
		instr  ssa.Instruction
		expect string
	}{
		{
			instrs[1],
			fmt.Sprintf(`%s/main.go:19:5`, pathInstrPos),
		},
		{
			instrs[4],
			fmt.Sprintf(`%s/main.go:22:11`, pathInstrPos),
		},
		{
			instrs[8],
			fmt.Sprintf(`%s/main.go:26:4`, pathInstrPos),
		},
		{
			instrs[13],
			fmt.Sprintf(`%s/main.go:30:19`, pathInstrPos),
		},
		{
			instrs[16],
			fmt.Sprintf(`%s/main.go:35:2`, pathInstrPos),
		},
		{
			instrs[17],
			fmt.Sprintf(`%s/main.go:35:5`, pathInstrPos),
		},
	}

	for idx, c := range cases {
		pos := usedtype.InstrPosition(fset, c.instr)
		require.Equal(t, c.expect, pos.String(), idx)
	}
}
