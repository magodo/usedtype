## Introduction

`usedtype` is a tool to print the used named type (including structure and interface whose implementers are structure) and its members (recursively) in one or more Go packages.

## Usage

```shell
usedtype -p <def pkg pattern> [options] <search package pattern>
  -callgraph string
        Whether to enable callgraph based analysis, can be one of: "", "cha", "static"
  -d    Whether to show debug log
  -p string
        The regexp pattern of import path of the package where the named types are defined.
  -v    Whether to output the lines of code for each field usage
```

### Callgraph Construction Method

Note that [`cha`](https://pkg.go.dev/golang.org/x/tools@v0.0.0-20210102185154-773b96fafca2/go/callgraph/cha) type tends to be quite time consuming, and the result of current process used in this tool might be similar as no callgraph analysis at all (i.e. `""`) if your code has dynamic dispatches everywhere. Whilst the benefit of `cha` (and `""`) are guaranteed to be "sound" (superset of "truth").

On the otherhand, [`static`](https://pkg.go.dev/golang.org/x/tools@v0.0.0-20210102185154-773b96fafca2/go/callgraph/static) type is fast, but the analysis result only takes [static calls](https://pkg.go.dev/golang.org/x/tools/go/ssa#CallCommon) into considerations. In which case, the [builtin function call and function variable](c and d case in "call" mode of SSA CallCommon section) and the [method call happens on interface type]("invoke" mode of SSA CallCommon section) will not be taken into consideration. This means the result might be "complete" (subset of "truth"). 

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
