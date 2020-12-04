package usedtype

import (
	"fmt"
	"go/types"
	"reflect"
	"strings"
)

type StructField struct {
	base  *types.Struct // This is always an underlying type of a Named type, which is canonical.
	index int
}

func (u StructField) Exported() bool {
	return u.base.Field(u.index).Exported()
}

func (u StructField) DereferenceRElem() types.Type {
	return DereferenceRElem(u.base.Field(u.index).Type())
}

func (u StructField) IsElemUnderlyingNamedStructOrInterface() bool {
	t := u.base.Field(u.index).Type()
	return IsElemUnderlyingNamedStructOrInterface(t)
}

func (u StructField) IsElemUnderlyingNamedInterface() bool {
	t := u.base.Field(u.index).Type()
	return IsElemUnderlyingNamedInterface(t)
}

func (u StructField) String() string {
	fieldName := u.base.Field(u.index).Name()
	tag := reflect.StructTag(u.base.Tag(u.index))
	jsonTag := tag.Get("json")
	idx := strings.Index(jsonTag, ",")
	var jsonTagName string
	if idx == -1 {
		jsonTagName = jsonTag
	} else {
		jsonTagName = jsonTag[:idx]
		if jsonTagName == "" {
			jsonTagName = u.base.Field(u.index).Name()
		}
	}

	return fmt.Sprintf("%s (%s)", fieldName, jsonTagName)
}
