// freemarker.go - FreeMarker template engine in golang.
// Copyright (C) 2017, b3log.org & hacpai.com
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package parse

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

var textFormat = "%s" // Changed to "%q" in tests for better error messages.

// A Node is an element in the parse tree. The interface is trivial.
// The interface contains an unexported method so that only
// types local to this package can satisfy it.
type Node interface {
	Type() NodeType
	String() string
	// Copy does a deep copy of the Node and all its components.
	// To avoid type assertions, some XxxNodes also have specialized
	// CopyXxx methods that return *XxxNode.
	Copy() Node
	Position() Pos // byte position of start of node in full original input string
	// tree returns the containing *Tree.
	// It is unexported so all implementations of Node are in this package.
	tree() *Tree
}

// NodeType identifies the type of a parse tree node.
type NodeType int

func (p Pos) Position() Pos {
	return p
}

// Type returns itself and provides an easy default implementation
// for embedding in a Node. Embedded in all non-trivial Nodes.
func (t NodeType) Type() NodeType {
	return t
}

const (
	NodeText       NodeType = iota // plain text
	NodeIf                         // if directive
	NodeBool                       // boolean constant
	NodeChain                      // sequence of field accesses
	NodeExpression                 // expression
	nodeElse                       // else action. Not added to tree
	nodeEnd                        // end action. Not added to tree
	NodeIdentifier                 // identifier
	NodeContent                    // list of Nodes
	NodeNil                        // untyped nil constant
	NodeNumber                     // numerical constant
	NodeList                       // list directive
	NodeString                     // string constant
	NodeTemplate                   // template invocation action
	NodeVariable                   // $ variable
)

// Nodes.

// ContentNode holds a sequence of nodes.
type ContentNode struct {
	NodeType
	Pos
	tr    *Tree
	Nodes []Node // element nodes in lexical order
}

func (t *Tree) newContent(pos Pos) *ContentNode {
	return &ContentNode{tr: t, NodeType: NodeContent, Pos: pos}
}

func (c *ContentNode) append(n Node) {
	c.Nodes = append(c.Nodes, n)
}

func (c *ContentNode) tree() *Tree {
	return c.tr
}

func (c *ContentNode) String() string {
	b := &bytes.Buffer{}

	for _, n := range c.Nodes {
		fmt.Fprint(b, n)
	}

	return b.String()
}

func (c *ContentNode) CopyContent() *ContentNode {
	if c == nil {
		return c
	}

	n := c.tr.newContent(c.Pos)
	for _, elem := range c.Nodes {
		n.append(elem.Copy())
	}

	return n
}

func (c *ContentNode) Copy() Node {
	return c.CopyContent()
}

// TextNode holds plain text.
type TextNode struct {
	NodeType
	Pos
	tr   *Tree
	Text []byte // The text; may span newlines.
}

func (t *Tree) newText(pos Pos, text string) *TextNode {
	return &TextNode{tr: t, NodeType: NodeText, Pos: pos, Text: []byte(text)}
}

func (t *TextNode) String() string {
	return fmt.Sprintf(textFormat, t.Text)
}

func (t *TextNode) tree() *Tree {
	return t.tr
}

func (t *TextNode) Copy() Node {
	return &TextNode{tr: t.tr, NodeType: NodeText, Pos: t.Pos, Text: append([]byte{}, t.Text...)}
}

type ExpressionNode struct {
	NodeType
	Pos
	tr       *Tree
	operator itemType
	Nodes    []Node // almost two nodes in case of binary expression, such as "a+b"
}

func (t *Tree) newExpression(pos Pos, optr itemType) *ExpressionNode {
	return &ExpressionNode{tr: t, NodeType: NodeExpression, Pos: pos, operator: optr}
}

func (c *ExpressionNode) append(node Node) {
	c.Nodes = append(c.Nodes, node)
}

func (c *ExpressionNode) String() string {
	s := ""
	for i, node := range c.Nodes {
		if i > 0 {
			s += c.operator.String()
		}
		if node, ok := node.(*ExpressionNode); ok {
			s += "(" + node.String() + ")"
			continue
		}
		s += node.String()
	}
	return s
}

func (c *ExpressionNode) tree() *Tree {
	return c.tr
}

func (c *ExpressionNode) CopyExpr() *ExpressionNode {
	if c == nil {
		return c
	}
	n := c.tr.newExpression(c.Pos, c.operator)
	for _, c := range c.Nodes {
		n.append(c.Copy())
	}
	return n
}

func (c *ExpressionNode) Copy() Node {
	return c.CopyExpr()
}

// IdentifierNode holds an identifier.
type IdentifierNode struct {
	NodeType
	Pos
	tr    *Tree
	Ident string // The identifier's name.
}

func (t *Tree) newIdentifier(pos Pos, ident string) *IdentifierNode {
	return &IdentifierNode{tr: t, NodeType: NodeIdentifier, Pos: pos, Ident: ident}
}

func (i *IdentifierNode) String() string {
	return i.Ident
}

func (i *IdentifierNode) tree() *Tree {
	return i.tr
}

func (i *IdentifierNode) Copy() Node {
	return &IdentifierNode{tr: i.tr, NodeType: i.NodeType, Pos: i.Pos, Ident: i.Ident}
}

// VariableNode holds a list of variable names, possibly with chained field
// accesses. The dollar sign is part of the (first) name.
type VariableNode struct {
	NodeType
	Pos
	tr    *Tree
	Ident []string // Variable name and fields in lexical order.
}

func (t *Tree) newVariable(pos Pos, ident string) *VariableNode {
	return &VariableNode{tr: t, NodeType: NodeVariable, Pos: pos, Ident: strings.Split(ident, ".")}
}

func (v *VariableNode) String() string {
	s := ""
	for i, id := range v.Ident {
		if i > 0 {
			s += "."
		}
		s += id
	}
	return s
}

func (v *VariableNode) tree() *Tree {
	return v.tr
}

func (v *VariableNode) Copy() Node {
	return &VariableNode{tr: v.tr, NodeType: NodeVariable, Pos: v.Pos, Ident: append([]string{}, v.Ident...)}
}

// NilNode holds the special identifier 'nil' representing an untyped nil constant.
type NilNode struct {
	NodeType
	Pos
	tr *Tree
}

func (t *Tree) newNil(pos Pos) *NilNode {
	return &NilNode{tr: t, NodeType: NodeNil, Pos: pos}
}

func (n *NilNode) Type() NodeType {
	// Override method on embedded NodeType for API compatibility.
	// TODO: Not really a problem; could change API without effect but
	// api tool complains.
	return NodeNil
}

func (n *NilNode) String() string {
	return "nil"
}

func (n *NilNode) tree() *Tree {
	return n.tr
}

func (n *NilNode) Copy() Node {
	return n.tr.newNil(n.Pos)
}

// ChainNode holds a term followed by a chain of field accesses (identifier starting with '.').
// The names may be chained ('.x.y').
// The periods are dropped from each ident.
type ChainNode struct {
	NodeType
	Pos
	tr    *Tree
	Node  Node
	Field []string // The identifiers in lexical order.
}

func (t *Tree) newChain(pos Pos, node Node) *ChainNode {
	return &ChainNode{tr: t, NodeType: NodeChain, Pos: pos, Node: node}
}

// Add adds the named field (which should start with a period) to the end of the chain.
func (c *ChainNode) Add(field string) {
	if len(field) == 0 || field[0] != '.' {
		panic("no dot in field")
	}
	field = field[1:] // Remove leading dot.
	if field == "" {
		panic("empty field")
	}
	c.Field = append(c.Field, field)
}

func (c *ChainNode) String() string {
	s := c.Node.String()
	if _, ok := c.Node.(*ExpressionNode); ok {
		s = "(" + s + ")"
	}
	for _, field := range c.Field {
		s += "." + field
	}
	return s
}

func (c *ChainNode) tree() *Tree {
	return c.tr
}

func (c *ChainNode) Copy() Node {
	return &ChainNode{tr: c.tr, NodeType: NodeChain, Pos: c.Pos, Node: c.Node, Field: append([]string{}, c.Field...)}
}

// BoolNode holds a boolean constant.
type BoolNode struct {
	NodeType
	Pos
	tr   *Tree
	True bool // The value of the boolean constant.
}

func (t *Tree) newBool(pos Pos, true bool) *BoolNode {
	return &BoolNode{tr: t, NodeType: NodeBool, Pos: pos, True: true}
}

func (b *BoolNode) String() string {
	if b.True {
		return "true"
	}

	return "false"
}

func (b *BoolNode) tree() *Tree {
	return b.tr
}

func (b *BoolNode) Copy() Node {
	return b.tr.newBool(b.Pos, b.True)
}

// NumberNode holds a number: signed or unsigned integer, float, or complex.
// The value is parsed and stored under all the types that can represent the value.
// This simulates in a small amount of code the behavior of Go's ideal constants.
type NumberNode struct {
	NodeType
	Pos
	tr         *Tree
	IsInt      bool       // Number has an integral value.
	IsUint     bool       // Number has an unsigned integral value.
	IsFloat    bool       // Number has a floating-point value.
	IsComplex  bool       // Number is complex.
	Int64      int64      // The signed integer value.
	Uint64     uint64     // The unsigned integer value.
	Float64    float64    // The floating-point value.
	Complex128 complex128 // The complex value.
	Text       string     // The original textual representation from the input.
}

func (t *Tree) newNumber(pos Pos, text string, typ itemType) (*NumberNode, error) {
	n := &NumberNode{tr: t, NodeType: NodeNumber, Pos: pos, Text: text}
	switch typ {
	case itemCharConstant:
		rune, _, tail, err := strconv.UnquoteChar(text[1:], text[0])
		if err != nil {
			return nil, err
		}
		if tail != "'" {
			return nil, fmt.Errorf("malformed character constant: %s", text)
		}
		n.Int64 = int64(rune)
		n.IsInt = true
		n.Uint64 = uint64(rune)
		n.IsUint = true
		n.Float64 = float64(rune) // odd but those are the rules.
		n.IsFloat = true
		return n, nil
		//	case itemComplex:
		//		// fmt.Sscan can parse the pair, so let it do the work.
		//		if _, err := fmt.Sscan(text, &n.Complex128); err != nil {
		//			return nil, err
		//		}
		//		n.IsComplex = true
		//		n.simplifyComplex()
		//		return n, nil
	}
	// Imaginary constants can only be complex unless they are zero.
	if len(text) > 0 && text[len(text)-1] == 'i' {
		f, err := strconv.ParseFloat(text[:len(text)-1], 64)
		if err == nil {
			n.IsComplex = true
			n.Complex128 = complex(0, f)
			n.simplifyComplex()
			return n, nil
		}
	}
	// Do integer test first so we get 0x123 etc.
	u, err := strconv.ParseUint(text, 0, 64) // will fail for -0; fixed below.
	if err == nil {
		n.IsUint = true
		n.Uint64 = u
	}
	i, err := strconv.ParseInt(text, 0, 64)
	if err == nil {
		n.IsInt = true
		n.Int64 = i
		if i == 0 {
			n.IsUint = true // in case of -0.
			n.Uint64 = u
		}
	}
	// If an integer extraction succeeded, promote the float.
	if n.IsInt {
		n.IsFloat = true
		n.Float64 = float64(n.Int64)
	} else if n.IsUint {
		n.IsFloat = true
		n.Float64 = float64(n.Uint64)
	} else {
		f, err := strconv.ParseFloat(text, 64)
		if err == nil {
			// If we parsed it as a float but it looks like an integer,
			// it's a huge number too large to fit in an int. Reject it.
			if !strings.ContainsAny(text, ".eE") {
				return nil, fmt.Errorf("integer overflow: %q", text)
			}
			n.IsFloat = true
			n.Float64 = f
			// If a floating-point extraction succeeded, extract the int if needed.
			if !n.IsInt && float64(int64(f)) == f {
				n.IsInt = true
				n.Int64 = int64(f)
			}
			if !n.IsUint && float64(uint64(f)) == f {
				n.IsUint = true
				n.Uint64 = uint64(f)
			}
		}
	}
	if !n.IsInt && !n.IsUint && !n.IsFloat {
		return nil, fmt.Errorf("illegal number syntax: %q", text)
	}
	return n, nil
}

// simplifyComplex pulls out any other types that are represented by the complex number.
// These all require that the imaginary part be zero.
func (n *NumberNode) simplifyComplex() {
	n.IsFloat = imag(n.Complex128) == 0
	if n.IsFloat {
		n.Float64 = real(n.Complex128)
		n.IsInt = float64(int64(n.Float64)) == n.Float64
		if n.IsInt {
			n.Int64 = int64(n.Float64)
		}
		n.IsUint = float64(uint64(n.Float64)) == n.Float64
		if n.IsUint {
			n.Uint64 = uint64(n.Float64)
		}
	}
}

func (n *NumberNode) String() string {
	return n.Text
}

func (n *NumberNode) tree() *Tree {
	return n.tr
}

func (n *NumberNode) Copy() Node {
	nn := new(NumberNode)
	*nn = *n // Easy, fast, correct.
	return nn
}

// StringNode holds a string constant.
type StringNode struct {
	NodeType
	Pos
	tr   *Tree
	Text string // The string, after quote processing.
}

func (t *Tree) newString(pos Pos, text string) *StringNode {
	return &StringNode{tr: t, NodeType: NodeString, Pos: pos, Text: text}
}

func (s *StringNode) String() string {
	return s.Text
}

func (s *StringNode) tree() *Tree {
	return s.tr
}

func (s *StringNode) Copy() Node {
	return s.tr.newString(s.Pos, s.Text)
}

// endNode represents an </# directive.
// It does not appear in the final parse tree.
type endNode struct {
	NodeType
	Pos
	tr         *Tree
	identifier string
}

func (t *Tree) newEnd(pos Pos, iden string) *endNode {
	return &endNode{tr: t, NodeType: nodeEnd, Pos: pos, identifier: iden}
}

func (e *endNode) String() string {
	return "</#" + e.identifier + ">"
}

func (e *endNode) tree() *Tree {
	return e.tr
}

func (e *endNode) Copy() Node {
	return e.tr.newEnd(e.Pos, e.identifier)
}

// elseNode represents an {{else}} action. Does not appear in the final tree.
type elseNode struct {
	NodeType
	Pos
	tr *Tree
}

func (t *Tree) newElse(pos Pos) *elseNode {
	return &elseNode{tr: t, NodeType: nodeElse, Pos: pos}
}

func (e *elseNode) Type() NodeType {
	return nodeElse
}

func (e *elseNode) String() string {
	return "{{else}}"
}

func (e *elseNode) tree() *Tree {
	return e.tr
}

func (e *elseNode) Copy() Node {
	return e.tr.newElse(e.Pos)
}

// IfNode represents a <#if> directive.
type IfNode struct {
	NodeType
	Pos
	tr          *Tree
	Expr        *ExpressionNode
	Content     *ContentNode
	ElseContent *ContentNode
}

func (t *Tree) newIf(pos Pos, expr *ExpressionNode, content, elseContent *ContentNode) *IfNode {
	return &IfNode{tr: t, NodeType: NodeIf, Pos: pos,
		Expr: expr, Content: content, ElseContent: elseContent}
}

func (ifNode *IfNode) String() string {
	return fmt.Sprintf("<#if %s>%s</#if>", ifNode.Expr, ifNode.Content)
}

func (ifNode *IfNode) tree() *Tree {
	return ifNode.tr
}

func (i *IfNode) Copy() Node {
	return i.tr.newIf(i.Pos, i.Expr.CopyExpr(), i.Content.CopyContent(), i.ElseContent.CopyContent())
}

// ListNode represents a <#list> directive.
type ListNode struct {
	NodeType
	Pos
	tr          *Tree
	Expr        *ExpressionNode
	Content     *ContentNode
	ElseContent *ContentNode
}

func (t *Tree) newList(pos Pos, expr *ExpressionNode, content *ContentNode, elseContent *ContentNode) *ListNode {
	return &ListNode{tr: t, NodeType: NodeList, Pos: pos,
		Expr: expr, Content: content, ElseContent: elseContent}
}

func (t *ListNode) String() string {
	return fmt.Sprintf("<#list %s>%s</#list>", t.Expr, t.Content)
}

func (t *ListNode) tree() *Tree {
	return t.tr
}

func (l *ListNode) Copy() Node {
	return l.tr.newList(l.Pos, l.Expr.CopyExpr(), l.Content.CopyContent(), l.ElseContent.CopyContent())
}
