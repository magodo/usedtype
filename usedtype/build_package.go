package usedtype

import (
	"errors"
	"fmt"

	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/callgraph/cha"
	"golang.org/x/tools/go/callgraph/rta"
	"golang.org/x/tools/go/callgraph/static"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/pointer"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

type CallGraphType string

const (
	CallGraphTypeStatic CallGraphType = "static"
	CallGraphTypeCha                  = "cha"
	CallGraphTypeRta                  = "rta"
	CallGraphTypePta                  = "pta"
	CallGraphTypeNA                   = ""
)

// BuildPackages accept the process argument and feed it to the packages.Load() to build
// both packages.Package and usedtype.Package(s) with a whole program build.
func BuildPackages(dir string, args []string, callgraphType CallGraphType) ([]*packages.Package, []*ssa.Package, *callgraph.Graph, error) {
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
	var graph *callgraph.Graph
	switch callgraphType {
	case CallGraphTypeStatic:
		graph = static.CallGraph(prog)
	case CallGraphTypeCha:
		graph = cha.CallGraph(prog)
	case CallGraphTypeRta:
		mains, err := mainPackages(prog.AllPackages())
		if err != nil {
			return nil, nil, nil, err
		}
		var roots []*ssa.Function
		for _, main := range mains {
			roots = append(roots, main.Func("init"), main.Func("main"))
		}
		rtares := rta.Analyze(roots, true)
		graph = rtares.CallGraph
	case CallGraphTypePta:
		mains, err := mainPackages(prog.AllPackages())
		if err != nil {
			return nil, nil, nil, err
		}
		config := &pointer.Config{
			Mains:          mains,
			BuildCallGraph: true,
		}
		ptares, err := pointer.Analyze(config)
		if err != nil {
			return nil, nil, nil, err
		}
		graph = ptares.CallGraph
	case CallGraphTypeNA:
		// do nothing
	default:
		return nil, nil, nil, fmt.Errorf("invalid call graph type: %s", callgraphType)
	}

	return pkgs, ssapkgs, graph, nil
}

// mainPackages returns the main packages to analyze.
// Each resulting package is named "main" and has a main function.
func mainPackages(pkgs []*ssa.Package) ([]*ssa.Package, error) {
	var mains []*ssa.Package
	for _, p := range pkgs {
		if p != nil && p.Pkg.Name() == "main" && p.Func("main") != nil {
			mains = append(mains, p)
		}
	}
	if len(mains) == 0 {
		return nil, fmt.Errorf("no main packages")
	}
	return mains, nil
}
