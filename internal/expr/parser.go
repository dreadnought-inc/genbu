package expr

import "fmt"

// Node represents an AST node.
type Node interface {
	nodeType() string
}

// FuncCall represents a function call with arguments.
type FuncCall struct {
	Name string
	Args []Node
}

func (f *FuncCall) nodeType() string { return "func" }

// VarRef represents a variable reference.
type VarRef struct {
	Name string
}

func (v *VarRef) nodeType() string { return "var" }

// StringLit represents a string literal.
type StringLit struct {
	Value string
}

func (s *StringLit) nodeType() string { return "string" }

// NumberLit represents a numeric literal.
type NumberLit struct {
	Value string
}

func (n *NumberLit) nodeType() string { return "number" }

type parser struct {
	tokens []token
	pos    int
}

func newParser(tokens []token) *parser {
	return &parser{tokens: tokens}
}

func (p *parser) peek() token {
	if p.pos >= len(p.tokens) {
		return token{typ: tokenEOF}
	}
	return p.tokens[p.pos]
}

func (p *parser) next() token {
	t := p.peek()
	p.pos++
	return t
}

func (p *parser) expect(typ tokenType) (token, error) {
	t := p.next()
	if t.typ != typ {
		return t, fmt.Errorf("expected token type %d, got %d (%q)", typ, t.typ, t.val)
	}
	return t, nil
}

// Parse parses the token stream into an AST node.
func (p *parser) Parse() (Node, error) {
	node, err := p.parseExpr()
	if err != nil {
		return nil, err
	}

	if p.peek().typ != tokenEOF {
		return nil, fmt.Errorf("unexpected token %q after expression", p.peek().val)
	}

	return node, nil
}

func (p *parser) parseExpr() (Node, error) {
	t := p.peek()

	switch t.typ {
	case tokenString:
		p.next()
		return &StringLit{Value: t.val}, nil

	case tokenNumber:
		p.next()
		return &NumberLit{Value: t.val}, nil

	case tokenIdent:
		p.next()
		// Look ahead: if '(' follows, it's a function call
		if p.peek().typ == tokenLParen {
			return p.parseFuncCall(t.val)
		}
		// Otherwise it's a variable reference
		return &VarRef{Name: t.val}, nil

	default:
		return nil, fmt.Errorf("unexpected token %q", t.val)
	}
}

func (p *parser) parseFuncCall(name string) (Node, error) {
	if _, err := p.expect(tokenLParen); err != nil {
		return nil, err
	}

	var args []Node

	// Handle empty argument list
	if p.peek().typ == tokenRParen {
		p.next()
		return &FuncCall{Name: name, Args: args}, nil
	}

	for {
		arg, err := p.parseExpr()
		if err != nil {
			return nil, fmt.Errorf("in arguments of %s: %w", name, err)
		}
		args = append(args, arg)

		if p.peek().typ == tokenComma {
			p.next()
			continue
		}
		break
	}

	if _, err := p.expect(tokenRParen); err != nil {
		return nil, fmt.Errorf("in function %s: %w", name, err)
	}

	return &FuncCall{Name: name, Args: args}, nil
}

// Parse parses an expression string into an AST.
func Parse(input string) (Node, error) {
	lex := newLexer(input)
	tokens, err := lex.tokenize()
	if err != nil {
		return nil, fmt.Errorf("lexer error: %w", err)
	}

	p := newParser(tokens)
	return p.Parse()
}

// CollectVarRefs walks the AST and returns all variable reference names.
func CollectVarRefs(node Node) []string {
	seen := make(map[string]bool)
	var refs []string

	var walk func(n Node)
	walk = func(n Node) {
		switch v := n.(type) {
		case *VarRef:
			if !seen[v.Name] {
				seen[v.Name] = true
				refs = append(refs, v.Name)
			}
		case *FuncCall:
			for _, arg := range v.Args {
				walk(arg)
			}
		}
	}

	walk(node)
	return refs
}
