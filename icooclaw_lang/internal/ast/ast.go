package ast

import (
	"github.com/issueye/icooclaw_lang/internal/lexer"
)

type Node interface {
	TokenLiteral() string
	String() string
}

type Stmt interface {
	Node
	statementNode()
}

type Expr interface {
	Node
	expressionNode()
}

type Program struct {
	Statements []Stmt
}

func (p *Program) TokenLiteral() string {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenLiteral()
	}
	return ""
}

func (p *Program) String() string {
	var out string
	for _, s := range p.Statements {
		out += s.String()
	}
	return out
}

type ExpressionStmt struct {
	Token lexer.Token
	Expr  Expr
}

func (es *ExpressionStmt) statementNode()       {}
func (es *ExpressionStmt) TokenLiteral() string { return es.Token.Literal }
func (es *ExpressionStmt) String() string {
	if es.Expr != nil {
		return es.Expr.String()
	}
	return ""
}

type LetStmt struct {
	Token lexer.Token
	Name  *Identifier
	Value Expr
}

func (ls *LetStmt) statementNode()       {}
func (ls *LetStmt) TokenLiteral() string { return ls.Token.Literal }
func (ls *LetStmt) String() string {
	if ls.Value != nil {
		return ls.Name.String() + " = " + ls.Value.String()
	}
	return ls.Name.String() + " ="
}

type ConstStmt struct {
	Token lexer.Token
	Name  *Identifier
	Value Expr
}

func (cs *ConstStmt) statementNode()       {}
func (cs *ConstStmt) TokenLiteral() string { return cs.Token.Literal }
func (cs *ConstStmt) String() string {
	return "const " + cs.Name.String() + " = " + cs.Value.String()
}

type ReturnStmt struct {
	Token       lexer.Token
	ReturnValue Expr
}

func (rs *ReturnStmt) statementNode()       {}
func (rs *ReturnStmt) TokenLiteral() string { return rs.Token.Literal }
func (rs *ReturnStmt) String() string {
	if rs.ReturnValue != nil {
		return "return " + rs.ReturnValue.String()
	}
	return "return"
}

type BreakStmt struct {
	Token lexer.Token
}

func (bs *BreakStmt) statementNode()       {}
func (bs *BreakStmt) TokenLiteral() string { return bs.Token.Literal }
func (bs *BreakStmt) String() string       { return "break" }

type ContinueStmt struct {
	Token lexer.Token
}

func (cs *ContinueStmt) statementNode()       {}
func (cs *ContinueStmt) TokenLiteral() string { return cs.Token.Literal }
func (cs *ContinueStmt) String() string       { return "continue" }

type BlockStmt struct {
	Token      lexer.Token
	Statements []Stmt
}

func (bs *BlockStmt) statementNode()       {}
func (bs *BlockStmt) TokenLiteral() string { return bs.Token.Literal }
func (bs *BlockStmt) String() string {
	var out string
	for _, s := range bs.Statements {
		out += s.String() + "\n"
	}
	return out
}

type IfStmt struct {
	Token       lexer.Token
	Condition   Expr
	Consequence *BlockStmt
	Alternative *BlockStmt
}

func (is *IfStmt) statementNode()       {}
func (is *IfStmt) TokenLiteral() string { return is.Token.Literal }
func (is *IfStmt) String() string {
	out := "if " + is.Condition.String() + " {\n" + is.Consequence.String() + "}"
	if is.Alternative != nil {
		out += " else {\n" + is.Alternative.String() + "}"
	}
	return out
}

type ForStmt struct {
	Token    lexer.Token
	Ident    *Identifier
	Iterable Expr
	Body     *BlockStmt
}

func (fs *ForStmt) statementNode()       {}
func (fs *ForStmt) TokenLiteral() string { return fs.Token.Literal }
func (fs *ForStmt) String() string {
	return "for " + fs.Ident.String() + " in " + fs.Iterable.String() + " {\n" + fs.Body.String() + "}"
}

type WhileStmt struct {
	Token     lexer.Token
	Condition Expr
	Body      *BlockStmt
}

func (ws *WhileStmt) statementNode()       {}
func (ws *WhileStmt) TokenLiteral() string { return ws.Token.Literal }
func (ws *WhileStmt) String() string {
	return "while " + ws.Condition.String() + " {\n" + ws.Body.String() + "}"
}

type FunctionStmt struct {
	Token  lexer.Token
	Name   *Identifier
	Params []*Identifier
	Body   *BlockStmt
}

func (fs *FunctionStmt) statementNode()       {}
func (fs *FunctionStmt) TokenLiteral() string { return fs.Token.Literal }
func (fs *FunctionStmt) String() string {
	params := ""
	for i, p := range fs.Params {
		if i > 0 {
			params += ", "
		}
		params += p.String()
	}
	return "fn " + fs.Name.String() + "(" + params + ") {\n" + fs.Body.String() + "}"
}

type MatchStmt struct {
	Token   lexer.Token
	Subject Expr
	Cases   []MatchCase
}

type MatchCase struct {
	Token    lexer.Token
	Patterns []Expr
	Guard    Expr
	Result   Expr
}

func (ms *MatchStmt) statementNode()       {}
func (ms *MatchStmt) expressionNode()      {}
func (ms *MatchStmt) TokenLiteral() string { return ms.Token.Literal }
func (ms *MatchStmt) String() string {
	out := "match " + ms.Subject.String() + " {\n"
	for _, c := range ms.Cases {
		out += c.String() + "\n"
	}
	out += "}"
	return out
}

func (mc *MatchCase) String() string {
	patterns := ""
	for i, p := range mc.Patterns {
		if i > 0 {
			patterns += " | "
		}
		patterns += p.String()
	}
	if mc.Guard != nil {
		patterns += " if " + mc.Guard.String()
	}
	return patterns + " -> " + mc.Result.String()
}

type TryStmt struct {
	Token      lexer.Token
	TryBlock   *BlockStmt
	CatchVar   *Identifier
	CatchBlock *BlockStmt
}

func (ts *TryStmt) statementNode()       {}
func (ts *TryStmt) TokenLiteral() string { return ts.Token.Literal }
func (ts *TryStmt) String() string {
	out := "try {\n" + ts.TryBlock.String() + "} catch " + ts.CatchVar.String() + " {\n" + ts.CatchBlock.String() + "}"
	return out
}

type ImportStmt struct {
	Token  lexer.Token
	Module Expr
	Names  []*Identifier
	Alias  *Identifier
}

func (is *ImportStmt) statementNode()       {}
func (is *ImportStmt) TokenLiteral() string { return is.Token.Literal }
func (is *ImportStmt) String() string {
	out := "import "
	if len(is.Names) > 0 {
		out += "{"
		for i, name := range is.Names {
			if i > 0 {
				out += ", "
			}
			out += name.String()
		}
		out += "} from " + is.Module.String()
	} else {
		out += is.Module.String()
	}
	if is.Alias != nil {
		out += " as " + is.Alias.String()
	}
	return out
}

type ExportStmt struct {
	Token lexer.Token
	Name  *Identifier
}

func (es *ExportStmt) statementNode()       {}
func (es *ExportStmt) TokenLiteral() string { return es.Token.Literal }
func (es *ExportStmt) String() string {
	return "export " + es.Name.String()
}

type GoStmt struct {
	Token lexer.Token
	Call  Expr
}

func (gs *GoStmt) statementNode()       {}
func (gs *GoStmt) TokenLiteral() string { return gs.Token.Literal }
func (gs *GoStmt) String() string {
	return "go " + gs.Call.String()
}

type Identifier struct {
	Token lexer.Token
	Value string
}

func (i *Identifier) expressionNode()      {}
func (i *Identifier) TokenLiteral() string { return i.Token.Literal }
func (i *Identifier) String() string       { return i.Value }

type IntegerLiteral struct {
	Token lexer.Token
	Value int64
}

func (il *IntegerLiteral) expressionNode()      {}
func (il *IntegerLiteral) TokenLiteral() string { return il.Token.Literal }
func (il *IntegerLiteral) String() string       { return il.Token.Literal }

type FloatLiteral struct {
	Token lexer.Token
	Value float64
}

func (fl *FloatLiteral) expressionNode()      {}
func (fl *FloatLiteral) TokenLiteral() string { return fl.Token.Literal }
func (fl *FloatLiteral) String() string       { return fl.Token.Literal }

type StringLiteral struct {
	Token lexer.Token
	Value string
}

func (sl *StringLiteral) expressionNode()      {}
func (sl *StringLiteral) TokenLiteral() string { return sl.Token.Literal }
func (sl *StringLiteral) String() string       { return "\"" + sl.Value + "\"" }

type BooleanLiteral struct {
	Token lexer.Token
	Value bool
}

func (bl *BooleanLiteral) expressionNode()      {}
func (bl *BooleanLiteral) TokenLiteral() string { return bl.Token.Literal }
func (bl *BooleanLiteral) String() string       { return bl.Token.Literal }

type NullLiteral struct {
	Token lexer.Token
}

func (nl *NullLiteral) expressionNode()      {}
func (nl *NullLiteral) TokenLiteral() string { return nl.Token.Literal }
func (nl *NullLiteral) String() string       { return "null" }

type PrefixExpr struct {
	Token    lexer.Token
	Operator string
	Right    Expr
}

func (pe *PrefixExpr) expressionNode()      {}
func (pe *PrefixExpr) TokenLiteral() string { return pe.Token.Literal }
func (pe *PrefixExpr) String() string {
	return "(" + pe.Operator + pe.Right.String() + ")"
}

type InfixExpr struct {
	Token    lexer.Token
	Left     Expr
	Operator string
	Right    Expr
}

func (ie *InfixExpr) expressionNode()      {}
func (ie *InfixExpr) TokenLiteral() string { return ie.Token.Literal }
func (ie *InfixExpr) String() string {
	return "(" + ie.Left.String() + " " + ie.Operator + " " + ie.Right.String() + ")"
}

type AssignExpr struct {
	Token lexer.Token
	Left  Expr
	Right Expr
}

func (ae *AssignExpr) expressionNode()      {}
func (ae *AssignExpr) TokenLiteral() string { return ae.Token.Literal }
func (ae *AssignExpr) String() string {
	return ae.Left.String() + " = " + ae.Right.String()
}

type CompoundAssignExpr struct {
	Token    lexer.Token
	Left     Expr
	Operator string
	Right    Expr
}

func (cae *CompoundAssignExpr) expressionNode()      {}
func (cae *CompoundAssignExpr) TokenLiteral() string { return cae.Token.Literal }
func (cae *CompoundAssignExpr) String() string {
	return cae.Left.String() + " " + cae.Operator + " " + cae.Right.String()
}

type CallExpr struct {
	Token     lexer.Token
	Function  Expr
	Arguments []Expr
}

func (ce *CallExpr) expressionNode()      {}
func (ce *CallExpr) TokenLiteral() string { return ce.Token.Literal }
func (ce *CallExpr) String() string {
	args := ""
	for i, a := range ce.Arguments {
		if i > 0 {
			args += ", "
		}
		args += a.String()
	}
	return ce.Function.String() + "(" + args + ")"
}

type ArrayLiteral struct {
	Token    lexer.Token
	Elements []Expr
}

func (al *ArrayLiteral) expressionNode()      {}
func (al *ArrayLiteral) TokenLiteral() string { return al.Token.Literal }
func (al *ArrayLiteral) String() string {
	elements := ""
	for i, e := range al.Elements {
		if i > 0 {
			elements += ", "
		}
		elements += e.String()
	}
	return "[" + elements + "]"
}

type HashLiteral struct {
	Token lexer.Token
	Pairs map[Expr]Expr
}

func (hl *HashLiteral) expressionNode()      {}
func (hl *HashLiteral) TokenLiteral() string { return hl.Token.Literal }
func (hl *HashLiteral) String() string {
	pairs := ""
	i := 0
	for k, v := range hl.Pairs {
		if i > 0 {
			pairs += ", "
		}
		pairs += k.String() + ": " + v.String()
		i++
	}
	return "{" + pairs + "}"
}

type IndexExpr struct {
	Token lexer.Token
	Left  Expr
	Index Expr
}

func (ie *IndexExpr) expressionNode()      {}
func (ie *IndexExpr) TokenLiteral() string { return ie.Token.Literal }
func (ie *IndexExpr) String() string {
	return ie.Left.String() + "[" + ie.Index.String() + "]"
}

type DotExpr struct {
	Token lexer.Token
	Left  Expr
	Right *Identifier
}

func (de *DotExpr) expressionNode()      {}
func (de *DotExpr) TokenLiteral() string { return de.Token.Literal }
func (de *DotExpr) String() string {
	return de.Left.String() + "." + de.Right.String()
}

type MethodCallExpr struct {
	Token     lexer.Token
	Object    Expr
	Method    *Identifier
	Arguments []Expr
}

func (mce *MethodCallExpr) expressionNode()      {}
func (mce *MethodCallExpr) TokenLiteral() string { return mce.Token.Literal }
func (mce *MethodCallExpr) String() string {
	args := ""
	for i, a := range mce.Arguments {
		if i > 0 {
			args += ", "
		}
		args += a.String()
	}
	return mce.Object.String() + "." + mce.Method.String() + "(" + args + ")"
}

type UnderscoreExpr struct {
	Token lexer.Token
}

func (ue *UnderscoreExpr) expressionNode()      {}
func (ue *UnderscoreExpr) TokenLiteral() string { return ue.Token.Literal }
func (ue *UnderscoreExpr) String() string       { return "_" }
