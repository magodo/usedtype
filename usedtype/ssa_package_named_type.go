package usedtype

import (
	"go/token"
	"go/types"
	"regexp"

	"golang.org/x/tools/go/packages"

	"golang.org/x/tools/go/ssa"
)

type NamedTypeAllocSet map[*types.Named]AllocSet

type Alloc struct {
	Instr    ssa.Instruction
	Position token.Position
}

type Allocs []Alloc

func (allocs Allocs) Len() int {
	return len(allocs)
}
func (allocs Allocs) Swap(i, j int) {
	allocs[i], allocs[j] = allocs[j], allocs[i]
}

func (allocs Allocs) Less(i, j int) bool {
	return allocs[i].Position.String() < allocs[j].Position.String()
}

type AllocSet map[Alloc]struct{}

type NamedTypeFilter func(pkg *packages.Package, t *types.Named) bool

// FindPackageNamedTypeAllocSet finds all the Alloc instructions among the SSA packages, whose underlying type is
// a named type that is defined in a package whose import path matches the "p" (pattern).
// If filter is given, it will further narrow down the result.
// TODO: we should eliminate the case that the alloc takes the value from a function variable.
func FindNamedTypeAllocSetInPackage(pkgs []*packages.Package, ssapkgs []*ssa.Package, p *regexp.Regexp, filter NamedTypeFilter) NamedTypeAllocSet {
	s := NamedTypeAllocSet{}
	for idx := range ssapkgs {
		ssapkg := ssapkgs[idx]
		pkg := pkgs[idx]

		var cb WalkInstrCallback
		cb = func(instr ssa.Instruction) {
			var nt *types.Named
			var ok bool

			switch instr := instr.(type) {
			case *ssa.Alloc:
				t := DereferenceRElem(instr.Type())
				nt, ok = t.(*types.Named)
				if !ok {
					return
				}
			case *ssa.MakeInterface:
				nt, ok = instr.Type().(*types.Named)
				if !ok {
					return
				}
			default:
				return
			}

			if nt.Obj() == nil {
				return
			}
			if nt.Obj().Pkg() == nil {
				return
			}
			if !p.MatchString(nt.Obj().Pkg().Path()) {
				return
			}
			if filter != nil && !filter(pkg, nt) {
				return
			}

			aset, ok := s[nt]
			if !ok {
				aset = AllocSet{}
				s[nt] = aset
			}
			aset[Alloc{
				Instr:    instr,
				Position: InstrPosition(pkg.Fset, instr),
			}] = struct{}{}
		}
		ssaTraversal := NewTraversal()
		ssaTraversal.WalkInPackage(ssapkg, cb, nil)
	}
	return s
}
