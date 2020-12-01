package usedtype

import (
	"go/types"
	"reflect"
	"strings"
)

type structField struct {
	index int
	base  *types.Named
}

type structFields []structField

func (field structField) Equal(ofield structField) bool {
	typeEqual := types.Identical(field.base, ofield.base)
	//return types.Identical(field.base, ofield.base) && field.index == ofield.index
	return typeEqual && field.index == ofield.index
}

func (field structField) Type() types.Type {
	return field.base.Underlying().(*types.Struct).Field(field.index).Type()
}

func (fields structFields) String() string {
	fieldStrs := []string{}
	for idx, field := range fields {
		if idx == 0 {
			named := field.base
			fieldStrs = append(fieldStrs, named.Obj().Name())
		}

		strct := field.base.Underlying().(*types.Struct)
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
