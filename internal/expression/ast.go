package expression

// Node represents a node in the Abstract Syntax Tree.
type Node interface {
	node()
}

// BinaryExpr represents a binary expression (&&, ||, ==, !=).
type BinaryExpr struct {
	Left     Node
	Operator TokenType
	Right    Node
}

func (b *BinaryExpr) node() {}

// UnaryExpr represents a unary expression (!).
type UnaryExpr struct {
	Operator TokenType
	Operand  Node
}

func (u *UnaryExpr) node() {}

// CallExpr represents a function call (e.g., success(), contains(a, b)).
type CallExpr struct {
	FuncName  string
	Arguments []Node
}

func (c *CallExpr) node() {}

// Identifier represents an identifier (e.g., env.VAR, steps.id.outcome).
type Identifier struct {
	Value string
}

func (i *Identifier) node() {}

// StringLiteral represents a string literal (e.g., 'value').
type StringLiteral struct {
	Value string
}

func (s *StringLiteral) node() {}

// BoolLiteral represents a boolean literal (true, false).
type BoolLiteral struct {
	Value bool
}

func (b *BoolLiteral) node() {}
