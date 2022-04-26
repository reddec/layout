package internal

import (
	"fmt"
	"strconv"
)

const (
	ContentDir   = "content"
	ManifestFile = "layout.yaml"
)

type Manifest struct {
	Title    string
	Prompts  []Prompt
	Computed []Computed
	Before   []Hook   // hook executed before generation
	After    []Hook   // hook executed after generation
	Ignore   []string // globs, filtered files will not be templated
}

type Prompt struct {
	Label   string // template
	Include string // template
	Var     string
	Type    VarType
	Options []string // allowed values, templated
	Default string   // template
	When    Condition
}

type Computed struct {
	Var   string
	Value interface{} // template only if value is string
	Type  VarType     // convert to this type if value is string, otherwise value used as-is
	When  Condition
}

type Hook struct {
	Run  string // templated, shell like (mvdan.cc/sh)
	When Condition
}

type VarType string

const (
	VarString VarType = "str"
	VarBool   VarType = "bool"
	VarInt    VarType = "int"
	VarFloat  VarType = "float"
	VarList   VarType = "list"
)

func (vt VarType) Parse(value string) (interface{}, error) {
	switch vt {
	case VarBool:
		return toBool(value), nil
	case VarList:
		return toList(value), nil
	case VarInt:
		return strconv.ParseInt(value, 10, 64)
	case VarFloat:
		return strconv.ParseFloat(value, 64)
	case "":
		fallthrough
	case VarString:
		return value, nil
	default:
		return nil, fmt.Errorf("unknown type %s", vt)
	}
}

type Condition string // tengo, by-default false
