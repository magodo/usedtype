package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/magodo/usedtype/usedtype"

	log "github.com/sirupsen/logrus"
)

const usage = `usedtype -p <def pkg pattern> [options] <search package pattern>`

var pattern = flag.String("p", "", "The regexp pattern of import path of the package where the named types are defined.")
var debug = flag.Bool("d", false, "Whether to show debug log")
var verbose = flag.Bool("v", false, "Whether to output the lines of code for each field usage")
var callGraphType = flag.String("callgraph", "",
	fmt.Sprintf(`Whether to enable callgraph based analysis, can be one of: "%s", "%s", "%s"`,
		usedtype.CallGraphTypeNA, usedtype.CallGraphTypeCha, usedtype.CallGraphTypeStatic))

func main() {
	log.Infof("Building packages (callgraph type: %s)...\n", *callGraphType)
	pkgs, ssapkgs, graph, err := usedtype.BuildPackages(".", flag.Args(), usedtype.CallGraphType(*callGraphType))
	if err != nil {
		log.Fatal(err)
	}

	usedtype.SetStructFieldUsageVerbose(*verbose)

	log.Debug("Finding package named type...")
	log.Infof("Finding package named type...")
	targetNamedTypeSet := usedtype.FindPackageNamedType(pkgs, *pattern, nil)
	log.Infof("Finding in-package structure direct usages...")
	directUsage := usedtype.FindInPackageStructureDirectUsage(pkgs, ssapkgs)
	log.Infof("Building struct full usages...")
	fus := usedtype.BuildStructFullUsages(directUsage, targetNamedTypeSet, graph)
	log.Infof("Finish building full usages")
	fmt.Println(fus)
}

func init() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "%s\n", usage)
		flag.PrintDefaults()
	}
	flag.Parse()
	if *pattern == "" {
		flag.Usage()
		os.Exit(1)
	}

	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	if *debug {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
}
