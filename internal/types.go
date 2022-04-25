package internal

type Manifest struct {
	Title    string
	Prompts  []Prompt
	Computed []Computed
	Before   []string // hook executed before, templated
	After    []string // hook executed after, templated
}

type Prompt struct {
	Label   string // template
	Include string // template
	Var     string
	Type    VarType
	Options []string // allowed values, templated, only for list or string type
	Default string   // template
	When    string   // tengo
}

type Computed struct {
	Var   string
	Value string // template
}

type VarType string

const (
	VarString VarType = "str"
	VarBool   VarType = "bool"
	VarInt    VarType = "int"
	VarFloat  VarType = "float"
	VarList   VarType = "list"
)
