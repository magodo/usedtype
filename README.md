## Introduction

`usedtype` is a tool to print the used named type (including structure and interface whose implementers are structure) and its members (recursively) in one or more Go packages.

> ❗ NOTE: The process used in this tool tends to be over-estimated (i.e. sound) given the goal is to calculate truth structure usage (coverage). In other words, some field in structure is reported as used, might be actually not used.

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

Note that [`cha`](https://pkg.go.dev/golang.org/x/tools@v0.0.0-20210102185154-773b96fafca2/go/callgraph/cha) type tends to be quite time consuming and the result might be similar as no callgraph analysis at all (i.e. `""`). Whilst `cha` and `""` are guaranteed to be "sound" (superset of "truth"). On the otherhand, [`static`](https://pkg.go.dev/golang.org/x/tools@v0.0.0-20210102185154-773b96fafca2/go/callgraph/static) type is fast, but the analysis result only takes static call edges into considerations, which means the result might be "complete" (subset of "truth").

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
