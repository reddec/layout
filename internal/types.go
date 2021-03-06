/*
Copyright 2022 Aleksandr Baryshnikov

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
	Version     string // minimal layout version (semver). Empty means any version
	Title       string // short description of what manifest doing, should be unique in multi-layouts repo
	Description string // full manifest description
	Delimiters  struct {
		Open  string
		Close string
	} // custom template delimiter for go templates, default is '{{' and '}}'
	Prompts  []Prompt
	Default  []Default  // computed values to define internal default values before processing state, useful in case of condition includes to prevent `undefined variable` error
	Computed []Computed // computed values used to calculate variables after user input
	Before   []Hook     // hook executed before generation
	After    []Hook     // hook executed after generation
	Ignore   []string   // globs, filtered files will not be templated
}

type Prompt struct {
	Label   string // template
	Include string // template
	Var     string
	Type    VarType
	Options []string    // allowed values, templated
	Default interface{} // template if not strings, array could be used for picking multiple default values (in case type is list)
	When    Condition
}

type Computed struct {
	Var   string
	Value interface{} // template only if value is string
	Type  VarType     // convert to this type if value is string, otherwise value used as-is
	When  Condition
}

type Default struct {
	Var   string
	Value interface{} // template only if value is string
	Type  VarType     // convert to this type if value is string, otherwise value used as-is
}

type Hook struct {
	Runnable `yaml:",inline"`
	Label    string // optional message which will displayed during execution. If nothing set, then nothing will be shown
	When     Condition
}

type Runnable struct {
	Run    string // templated, shell like (mvdan.cc/sh)
	Script string // path to script (executable), relative to manifest, content templated. It has limited support for shell execution, and designed for direct script invocation: <script> [args...]
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
