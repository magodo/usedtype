## Introduction

`usedtype` is a tool to print the used named type (including structure and interface whose implementers are structure) and its members (recursively) in one or more Go packages.

## Usage

```shell
usedtype -p <def pkg pattern> <search package pattern>                                    
  -p string                                                                               
        The regexp pattern of import path of the package where the named types are defined
```

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
