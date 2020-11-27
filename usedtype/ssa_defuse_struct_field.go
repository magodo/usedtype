package usedtype

import (
	"go/types"
	"reflect"
	"strings"
)

type structField struct {
	index int
	t     types.Type
}

type structFields []structField

func (fields structFields) String() string {
	var underlyingNamed func(t types.Type) *types.Named
	underlyingNamed = func(t types.Type) *types.Named {
		switch t := t.(type) {
		case *types.Named:
			return t
		case *types.Pointer:
			return underlyingNamed(t.Elem())
		default:
			panic("Not gonna happen")
		}
	}

	underlyingStruct := func(t types.Type) *types.Struct {
		named := underlyingNamed(t)
		switch v := named.Underlying().(type) {
		case *types.Struct:
			return v
		default:
			panic("Not gonna happen")
		}
	}

	fieldStrs := []string{}
	for idx, field := range fields {
		if idx == 0 {
			named := underlyingNamed(field.t)
			fieldStrs = append(fieldStrs, named.Obj().Name())
		}

		strct := underlyingStruct(field.t)
		tag := reflect.StructTag(strct.Tag(field.index))
		jsonTag := tag.Get("json")
		idx := strings.Index(jsonTag, ",")
		var fieldName string
		if idx == -1 {
			fieldName = jsonTag
		} else {
			fieldName = jsonTag[:idx]
			if fieldName == "" {
				fieldName = strct.Field(field.index).Name()
			}
		}

		// This field is ignored in json request
		if fieldName == "-" {
			continue
		}
		fieldStrs = append(fieldStrs, fieldName)
	}
	return strings.Join(fieldStrs, ".")
}
