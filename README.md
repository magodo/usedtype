## Introduction

`usedtype` is a tool to print the used named type (including structure and interface whose implementers are structure) and its members (recursively) in one or more Go packages.

## Usage

```shell
usedtype -p <def pkg pattern> [options] <search package pattern>
  -callgraph string
        Whether to enable callgraph based analysis, can be one of: "", "static", "cha", "rta", "pta"
        (Note that rta and pta require a whole program (main or test), and include only functions reachable from main)
  -d    Whether to show debug log
  -p string
        The regexp pattern of import path of the package where the named types are defined.
  -v    Whether to output the lines of code for each field usage
```

### Callgraph Construction Method

Speed: `"" > static > cha > rta > pta`
Precision: `"" < static (unsound) < cha < rta < pta`

Especially, [`static`](https://pkg.go.dev/golang.org/x/tools@v0.0.0-20210102185154-773b96fafca2/go/callgraph/static) only takes [static calls](https://pkg.go.dev/golang.org/x/tools/go/ssa#CallCommon) into considerations. In which case, the builtin function call and function variable (declared then set) (c and d case in "call" mode of SSA CallCommon section) and the method call happens on interface type("invoke" mode of SSA CallCommon section) will not be taken into consideration. This means the result might be "complete" (subset of "truth"). 

## Example

```shell
$ pwd                         
/media/storage/github/usedtype 

$ usedtype -p golang.org/x/tools/go/ssa ./...
                                              
golang.org/x/tools/go/ssa.Field               
    Field                                     
golang.org/x/tools/go/ssa.FieldAddr           
    Field                                     
golang.org/x/tools/go/ssa.Function            
    Params                                    
    Blocks                                    
    AnonFuncs                                 
golang.org/x/tools/go/ssa.Package             
    Members                                   
```

This shows which structures (and their fields) that are from `golang.org/x/tools/go/ssa` package are used in this project.
