package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/magodo/usedtype/usedtype"
)

const usage = `usedtype -p <def pkg pattern> [options] <search package pattern>`

var pattern = flag.String("p", "", "The regexp pattern of import path of the package where the named types are defined.")
var verbose = flag.Bool("v", false, "Whether to output the lines of code for each field usage")
var enableCallGraphAnalysis = flag.Bool("c", false, "Whether to enable callgraph based analysis")

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

	log.Printf("Building packages (callgraph enabled: %t)...\n", *enableCallGraphAnalysis)
	pkgs, ssapkgs, graph, err := usedtype.BuildPackages(".", flag.Args(), *enableCallGraphAnalysis)
	if err != nil {
		log.Fatal(err)
	}

	usedtype.SetStructFieldUsageVerbose(*verbose)

	log.Println("Finding package named type...")
	targetNamedTypeSet := usedtype.FindPackageNamedType(pkgs, *pattern, nil)
	log.Println("Finding in-package structure direct usages...")
	directUsage := usedtype.FindInPackageStructureDirectUsage(pkgs, ssapkgs)
	log.Println("Building struct full usages...")
	fus := usedtype.BuildStructFullUsages(directUsage, targetNamedTypeSet, graph)
	log.Println("Finish building full usages")
	fmt.Println(fus)
}
