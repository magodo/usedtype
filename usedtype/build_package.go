package usedtype

import (
	"errors"

	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/callgraph/cha"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

// BuildPackages accept the process argument and feed it to the packages.Load() to build
// both packages.Package and usedtype.Package(s) with a whole program build.
func BuildPackages(dir string, args []string) ([]*packages.Package, []*ssa.Package, *callgraph.Graph, error) {
	cfg := packages.Config{Dir: dir, Mode: packages.LoadAllSyntax}
	pkgs, err := packages.Load(&cfg, args...)
	if err != nil {
		return nil, nil, nil, err
	}

	// Stop if any package had errors.
	// This step is optional; without it, the previous step
	// will create SSA for only a subset of packages.
	if packages.PrintErrors(pkgs) > 0 {
		return nil, nil, nil, errors.New("packages contain errors")
	}

	// Build SSA for the specified "pkgs" and their dependencies.
	// The returned ssapkgs is the corresponding SSA Package of the specified "pkgs".
	prog, ssapkgs := ssautil.AllPackages(pkgs, 0)
	prog.Build()

	// Build Callgraph
	graph := cha.CallGraph(prog)

	return pkgs, ssapkgs, graph, nil
}
