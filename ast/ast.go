package ast

import "waiig/token"

type Node interface {
	TokenLiteral() string
}

// Statements don't produce values(think of `let a = 1;`)
type Statement interface {
	Node
	// These private methods don't seem to do anything other than provide some type safety around types of Nodes,
	// this is because Go doesn't have an `implements` keyword, implementing an interface in Go is done by simply
	// making the type have all the methods of the interface, hence this statementNode() and expressionNode() methods
	// force the type to override and "pick" either a Statement or an Expression, since they both of these types
	// share the same methods other than these private method it could because tricky to understand at first glance
	statementNode()
}

// Expressions produce values(think of `add(1, 2);`)
type Expression interface {
	Node
	expressionNode()
}

type Program struct {
	Statements []Statement
}

func (p *Program) TokenLiteral() string {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenLiteral()
	} else {
		return ""
	}
}

type LetStatement struct {
	Token token.Token
	Name  *Identifier
	Value Expression
}

func (ls *LetStatement) statementNode()       {}
func (ls *LetStatement) TokenLiteral() string { return ls.Token.Literal }

// Identifier could classify as both Expressions and Statements, seeing that in `let a = 1;`, `a` identifier doesn't produce
// a value, so it could be considered a statement, but in `let b = a;`, `a` identifier produces a value, so it could be
// considered an expression, to simplify, so we don't have two types of Identifier nodes, we're merging both into this one
type Identifier struct {
	Token token.Token
	Value string
}

func (i *Identifier) expressionNode()      {}
func (i *Identifier) TokenLiteral() string { return i.Token.Literal }
