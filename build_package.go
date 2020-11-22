package main

import (
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
	"log"
	"os"
)

func buildPackages() ([]*packages.Package, []*ssa.Package) {
	cfg := packages.Config{Mode: packages.LoadAllSyntax}
	pkgs, err := packages.Load(&cfg, os.Args[1:]...)
	if err != nil {
		log.Fatal(err)
	}

	// Stop if any package had errors.
	// This step is optional; without it, the previous step
	// will create SSA for only a subset of packages.
	if packages.PrintErrors(pkgs) > 0 {
		log.Fatalf("packages contain errors")
	}

	// Build SSA for the specified "pkgs" and their dependencies.
	// The returned ssapkgs is the corresponding SSA Package of the specified "pkgs".
	prog, ssapkgs := ssautil.AllPackages(pkgs, 0)
	prog.Build()
	return pkgs, ssapkgs
}
