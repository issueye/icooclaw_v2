package object

import (
	"fmt"
	"strings"

	"github.com/issueye/icooclaw_lang/internal/ast"
)

type ObjectType string

const (
	INTEGER_OBJ  = "INTEGER"
	FLOAT_OBJ    = "FLOAT"
	STRING_OBJ   = "STRING"
	BOOLEAN_OBJ  = "BOOLEAN"
	NULL_OBJ     = "NULL"
	ARRAY_OBJ    = "ARRAY"
	HASH_OBJ     = "HASH"
	FUNCTION_OBJ = "FUNCTION"
	BUILTIN_OBJ  = "BUILTIN"
	RETURN_OBJ   = "RETURN"
	ERROR_OBJ    = "ERROR"
	BREAK_OBJ    = "BREAK"
	CONTINUE_OBJ = "CONTINUE"
)

type Object interface {
	Type() ObjectType
	Inspect() string
}

type Integer struct {
	Value int64
}

func (i *Integer) Type() ObjectType { return INTEGER_OBJ }
func (i *Integer) Inspect() string  { return fmt.Sprintf("%d", i.Value) }

type Float struct {
	Value float64
}

func (f *Float) Type() ObjectType { return FLOAT_OBJ }
func (f *Float) Inspect() string  { return fmt.Sprintf("%g", f.Value) }

type String struct {
	Value string
}

func (s *String) Type() ObjectType { return STRING_OBJ }
func (s *String) Inspect() string  { return s.Value }

type Boolean struct {
	Value bool
}

func (b *Boolean) Type() ObjectType { return BOOLEAN_OBJ }
func (b *Boolean) Inspect() string  { return fmt.Sprintf("%t", b.Value) }

type Null struct{}

func (n *Null) Type() ObjectType { return NULL_OBJ }
func (n *Null) Inspect() string  { return "null" }

type Array struct {
	Elements []Object
}

func (a *Array) Type() ObjectType { return ARRAY_OBJ }
func (a *Array) Inspect() string {
	elements := make([]string, len(a.Elements))
	for i, e := range a.Elements {
		elements[i] = e.Inspect()
	}
	return "[" + strings.Join(elements, ", ") + "]"
}

type HashPair struct {
	Key   Object
	Value Object
}

type Hash struct {
	Pairs map[string]HashPair
}

func (h *Hash) Type() ObjectType { return HASH_OBJ }
func (h *Hash) Inspect() string {
	pairs := make([]string, 0, len(h.Pairs))
	for _, pair := range h.Pairs {
		pairs = append(pairs, pair.Key.Inspect()+": "+pair.Value.Inspect())
	}
	return "{" + strings.Join(pairs, ", ") + "}"
}

type Function struct {
	Name   string
	Params []*ast.Identifier
	Body   *ast.BlockStmt
	Env    *Environment
}

func (f *Function) Type() ObjectType { return FUNCTION_OBJ }
func (f *Function) Inspect() string {
	params := make([]string, len(f.Params))
	for i, p := range f.Params {
		params[i] = p.String()
	}
	return "fn " + f.Name + "(" + strings.Join(params, ", ") + ") { ... }"
}

type BuiltinFunction func(env *Environment, args ...Object) Object

type Builtin struct {
	Fn BuiltinFunction
}

func (b *Builtin) Type() ObjectType { return BUILTIN_OBJ }
func (b *Builtin) Inspect() string  { return "builtin function" }

type Return struct {
	Value Object
}

func (r *Return) Type() ObjectType { return RETURN_OBJ }
func (r *Return) Inspect() string  { return r.Value.Inspect() }

type Error struct {
	Message string
	Line    int
}

func (e *Error) Type() ObjectType { return ERROR_OBJ }
func (e *Error) Inspect() string  { return fmt.Sprintf("ERROR: %s (line %d)", e.Message, e.Line) }

type Break struct{}

func (b *Break) Type() ObjectType { return BREAK_OBJ }
func (b *Break) Inspect() string  { return "break" }

type Continue struct{}

func (c *Continue) Type() ObjectType { return CONTINUE_OBJ }
func (c *Continue) Inspect() string  { return "continue" }

var (
	nullSingleton  = &Null{}
	trueSingleton  = &Boolean{Value: true}
	falseSingleton = &Boolean{Value: false}
)

func NullObject() *Null {
	return nullSingleton
}

func BoolObject(v bool) *Boolean {
	if v {
		return trueSingleton
	}
	return falseSingleton
}

func IsError(obj Object) bool {
	return obj != nil && obj.Type() == ERROR_OBJ
}

func IsReturn(obj Object) bool {
	return obj != nil && obj.Type() == RETURN_OBJ
}

func IsBreak(obj Object) bool {
	return obj != nil && obj.Type() == BREAK_OBJ
}

func IsContinue(obj Object) bool {
	return obj != nil && obj.Type() == CONTINUE_OBJ
}

func IsTruthy(obj Object) bool {
	switch obj := obj.(type) {
	case *Boolean:
		return obj.Value
	case *Null:
		return false
	case *Integer:
		return obj.Value != 0
	case *Float:
		return obj.Value != 0.0
	case *String:
		return obj.Value != ""
	default:
		return true
	}
}

func NewError(line int, format string, a ...interface{}) *Error {
	return &Error{Message: fmt.Sprintf(format, a...), Line: line}
}

func HashKey(obj Object) string {
	switch obj := obj.(type) {
	case *String:
		return obj.Value
	case *Integer:
		return fmt.Sprintf("%d", obj.Value)
	case *Boolean:
		return fmt.Sprintf("%t", obj.Value)
	default:
		return obj.Inspect()
	}
}

func HashFromObjects(values map[string]Object) *Hash {
	pairs := make(map[string]HashPair, len(values))
	for key, value := range values {
		keyObj := &String{Value: key}
		pairs[key] = HashPair{Key: keyObj, Value: value}
	}
	return &Hash{Pairs: pairs}
}
