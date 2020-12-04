package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/magodo/usedtype/usedtype"
)

const usage = `usedtype -p <def pkg pattern> <search package pattern>`

var pattern = flag.String("p", "", "The regexp pattern of import path of the package where the named types are defined.")

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "%s\n", usage)
		flag.PrintDefaults()
	}
	flag.Parse()
	if *pattern == "" {
		flag.Usage()
		os.Exit(1)
	}

	pkgs, ssapkgs, err := usedtype.BuildPackages(".", flag.Args())
	if err != nil {
		log.Fatal(err)
	}

	targetNamedTypeSet := usedtype.FindPackageNamedType(pkgs, *pattern, nil)
	directUsage := usedtype.FindInPackageStructureDirectUsage(pkgs, ssapkgs)
	fus := usedtype.BuildStructFullUsages(directUsage, targetNamedTypeSet)
	fmt.Println(fus)
}
