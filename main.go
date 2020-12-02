package main

import (
	"flag"
	"fmt"
	"go/types"
	"log"
	"os"

	"github.com/magodo/usedtype/usedtype"
	"golang.org/x/tools/go/packages"
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

	// Analyze all the target external packages and get a list of types.Object
	targetStructSets := usedtype.FindExternalPackageStruct(pkgs, *pattern, terraformSchemaTypeFilter)

	directUsage := usedtype.FindInPackageStructureDirectUsage(pkgs, ssapkgs)

	//for t := range targetStructSets {
	//	if u, ok := usage[t]; ok {
	//		fmt.Printf("===\n%s\n===\n%s\n", t.String(), u)
	//	}
	//}
	for k := range targetStructSets {
		fu := usedtype.BuildStructFullUsage(directUsage, k)
		fmt.Println(fu)
	}

	//
	//// Find all ssa def node of the current package.
	//ssadefs := usedtype.FindInPackageAllDefValue(pkgs, ssapkgs)
	//
	//var allOduChains usedtype.ODUChainCluster = map[ssa.Value]usedtype.ODUChains{}
	//for _, value := range ssadefs {
	//	allOduChains[value.Value] = usedtype.WalkODUChains(value.Value, ssapkgs, value.Fset)
	//}
	//
	//fmt.Println(allOduChains.String())
	//allOduChains.Pair()
	//
	//structNodes := usedtype.FindInPackageDefValueOfTargetStructType(ssapkgs, targetStructs)
	//for k, values := range structNodes {
	//	fmt.Println(k.TypeName)
	//	for _, v := range values {
	//		for _, chain := range allOduChains[v] {
	//			fmt.Println(chain.Fields())
	//		}
	//	}
	//}
}

func terraformSchemaTypeFilter(epkg *packages.Package, t *types.Named) bool {
	scope := epkg.Types.Scope()
	for _, topType := range scope.Names() {
		et := scope.Lookup(topType).Type()
		switch et := et.(type) {
		case *types.Named:
			var c, d *types.Func
			for i := 0; i < et.NumMethods(); i++ {
				m := et.Method(i)
				switch m.Name() {
				case "CreateOrUpdate",
					"Create":
					c = m
				case "Delete":
					d = m
				}
			}
			// Terraform only care resources that can be created and deleted.
			if c == nil || d == nil {
				continue
			}
			signature := c.Type().(*types.Signature)
			lastParam := signature.Params().At(signature.Params().Len() - 1)
			if types.Identical(lastParam.Type(), t) {
				return true
			}
		default:
			continue
		}
	}
	return false
}
