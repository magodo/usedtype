package usedtype

import (
	"go/types"

	"golang.org/x/tools/go/callgraph"
)

// CustomImplements checks whether type "v" implements a named interface "itf".
// Note that the user has to ensure the "itf"'s underlying type is an interface.
type CustomImplements func(v types.Type, itf *types.Named) bool

type StructFullBuildOption struct {
	// If non-nil, the struct full build process will further check the reachability based on the call graph when extending the properties.
	Callgraph *callgraph.Graph

	// If non-nil, it is used to check whether a type implement an interface, which affects the result that diverges structures from an interface during the usage build.
	// If this is not set, the default function used for this check is the `types.Implements()` defined in go/types package.
	// Note that in almost all the cases, you will leave it as nil.
	CustomImplements CustomImplements
}
