package internal

type Manifest struct {
	Title    string
	Prompts  []Prompt
	Computed []Computed
	Before   []Hook // hook executed before
	After    []Hook // hook executed after
}

type Prompt struct {
	Label   string // template
	Include string // template
	Var     string
	Type    VarType
	Options []string // allowed values, templated, only for list or string type
	Default string   // template
	When    Condition
}

type Computed struct {
	Var   string
	Value string // template
	When  Condition
}

type Hook struct {
	Run  string // templated
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

type Condition string // tengo, by-default false
