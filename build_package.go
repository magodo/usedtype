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

	// Create SSA packages for all well-typed packages.
	_, ssapkgs := ssautil.Packages(pkgs, 0)
	//prog, ssapkgs := ssautil.Packages(pkgs, ssa.GlobalDebug)

	for _, pkg := range ssapkgs {
		pkg.Build()
	}
	return pkgs, ssapkgs
}
