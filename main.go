package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/magodo/usedtype/usedtype"
)

const usage = `usedtype -p <external pkg pattern> <package>`

var pattern = flag.String("p", "", "The regexp pattern for package import path of the external package to scan the struct coverage.")

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "%s\n", usage)
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
