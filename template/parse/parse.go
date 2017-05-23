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

// Package parse builds parse trees for FreeMarker templates. Clients should use
// github.com/b3log/freemarker.go package to construct templates rather than
// this one, which provides shared internal data structures not intended for
// general use.
package parse

import (
	"bytes"
	"fmt"
	"runtime"
	"strconv"
	"strings"
)

// Tree is the representation of a single parsed template.
type Tree struct {
	Name      string       // name of the template represented by the tree
	ParseName string       // name of the top-level template during parsing, for error messages
	Root      *ContentNode // top-level root of the tree
	text      string       // text parsed to create the template (or its parent)
	lex       *lexer
	token     [3]item // three-token lookahead for parser
	peekCount int
	treeSet   map[string]*Tree
}

// Copy returns a copy of the Tree. Any parsing state is discarded.
func (t *Tree) Copy() *Tree {
	if t == nil {
		return nil
	}

	return &Tree{
		Name:      t.Name,
		ParseName: t.ParseName,
		Root:      t.Root.CopyContent(),
		text:      t.text,
	}
}

// Parse returns a map from template name to parse.Tree, created by parsing the
// templates described in the argument string. The top-level template will be
// given the specified name. If an error is encountered, parsing stops and an
// empty map is returned with the error.
func Parse(name, text string, funcs ...map[string]interface{}) (map[string]*Tree, error) {
	treeSet := make(map[string]*Tree)
	t := New(name)
	t.text = text
	_, err := t.Parse(text, treeSet, funcs...)

	return treeSet, err
}

// next returns the next token.
func (t *Tree) next() item {
	if t.peekCount > 0 {
		t.peekCount--
	} else {
		t.token[0] = t.lex.nextItem()
	}

	return t.token[t.peekCount]
}

// backup backs the input stream up one token.
func (t *Tree) backup() {
	t.peekCount++
}

// backup2 backs the input stream up two tokens.
// The zeroth token is already there.
func (t *Tree) backup2(t1 item) {
	t.token[1] = t1
	t.peekCount = 2
}

// backup3 backs the input stream up three tokens
// The zeroth token is already there.
func (t *Tree) backup3(t2, t1 item) {
	// Reverse order: we're pushing back.
	t.token[1] = t1
	t.token[2] = t2
	t.peekCount = 3
}

// peek returns but does not consume the next token.
func (t *Tree) peek() item {
	if t.peekCount > 0 {
		return t.token[t.peekCount-1]
	}
	t.peekCount = 1
	t.token[0] = t.lex.nextItem()

	return t.token[0]
}

// nextNonSpace returns the next non-space token.
func (t *Tree) nextNonSpace() (token item) {
	for {
		token = t.next()
		if token.typ != itemSpace {
			break
		}
	}

	return token
}

// peekNonSpace returns but does not consume the next non-space token.
func (t *Tree) peekNonSpace() (token item) {
	for {
		token = t.next()
		if token.typ != itemSpace {
			break
		}
	}
	t.backup()

	return token
}

// Parsing.

// New allocates a new parse tree with the given name.
func New(name string) *Tree {
	return &Tree{
		Name: name,
	}
}

// ErrorContext returns a textual representation of the location of the node in the input text.
// The receiver is only used when the node does not have a pointer to the tree inside,
// which can occur in old code.
func (t *Tree) ErrorContext(n Node) (location, context string) {
	pos := int(n.Position())
	tree := n.tree()
	if tree == nil {
		tree = t
	}
	text := tree.text[:pos]
	byteNum := strings.LastIndex(text, "\n")
	if byteNum == -1 {
		byteNum = pos // On first line.
	} else {
		byteNum++ // After the newline.
		byteNum = pos - byteNum
	}
	lineNum := 1 + strings.Count(text, "\n")
	context = n.String()
	if len(context) > 20 {
		context = fmt.Sprintf("%.20s...", context)
	}

	return fmt.Sprintf("%s:%d:%d", tree.ParseName, lineNum, byteNum), context
}

// errorf formats the error and terminates processing.
func (t *Tree) errorf(format string, args ...interface{}) {
	t.Root = nil
	format = fmt.Sprintf("template: %s:%d: %s", t.ParseName, t.token[0].line, format)
	panic(fmt.Errorf(format, args...))
}

// error terminates processing.
func (t *Tree) error(err error) {
	t.errorf("%s", err)
}

// expect consumes the next token and guarantees it has the required type.
func (t *Tree) expect(expected itemType, context string) item {
	token := t.nextNonSpace()
	if token.typ != expected {
		t.unexpected(token, context)
	}

	return token
}

// expectOneOf consumes the next token and guarantees it has one of the required types.
func (t *Tree) expectOneOf(expected1, expected2 itemType, context string) item {
	token := t.nextNonSpace()
	if token.typ != expected1 && token.typ != expected2 {
		t.unexpected(token, context)
	}

	return token
}

// unexpected complains about the token and terminates processing.
func (t *Tree) unexpected(token item, context string) {
	t.errorf("unexpected %s in %s", token, context)
}

// recover is the handler that turns panics into returns from the top level of Parse.
func (t *Tree) recover(errp *error) {
	e := recover()
	if e != nil {
		if _, ok := e.(runtime.Error); ok {
			panic(e)
		}
		if t != nil {
			t.lex.drain()
			t.stopParse()
		}
		*errp = e.(error)
	}
}

// startParse initializes the parser, using the lexer.
func (t *Tree) startParse(funcs []map[string]interface{}, lex *lexer, treeSet map[string]*Tree) {
	t.Root = nil
	t.lex = lex
	t.treeSet = treeSet
}

// stopParse terminates parsing.
func (t *Tree) stopParse() {
	t.lex = nil
	t.treeSet = nil
}

// Parse parses the template definition string to construct a representation of
// the template for execution. If either action delimiter string is empty, the
// default ("{{" or "}}") is used. Embedded template definitions are added to
// the treeSet map.
func (t *Tree) Parse(text string, treeSet map[string]*Tree, funcs ...map[string]interface{}) (tree *Tree, err error) {
	defer t.recover(&err)
	t.ParseName = t.Name
	t.startParse(funcs, lex(t.Name, text), treeSet)
	t.text = text
	t.parse()
	t.add()
	t.stopParse()
	return t, nil
}

// add adds tree to t.treeSet.
func (t *Tree) add() {
	tree := t.treeSet[t.Name]
	if tree == nil || IsEmptyTree(tree.Root) {
		t.treeSet[t.Name] = t
		return
	}
	if !IsEmptyTree(t.Root) {
		t.errorf("template: multiple definition of template %q", t.Name)
	}
}

// IsEmptyTree reports whether this tree (node) is empty of everything but space.
func IsEmptyTree(n Node) bool {
	switch n := n.(type) {
	case nil:
		return true
	case *IfNode:
	case *ContentNode:
		for _, node := range n.Nodes {
			if !IsEmptyTree(node) {
				return false
			}
		}
		return true
	case *ListNode:
	case *TextNode:
		return len(bytes.TrimSpace(n.Text)) == 0
	default:
		panic("unknown node: " + n.String())
	}
	return false
}

// parse is the top-level parser for a template, essentially the same
// as itemContent except it also parses <#include>, <#macro> directives.
// It runs to EOF.
func (t *Tree) parse() {
	t.Root = t.newContent(t.peek().pos)

	for t.peek().typ != itemEOF {
		//if t.peek().typ == itemLeftDirective {
		//	delim := t.next()
		//			if t.nextNonSpace().typ == itemDefine {
		//				newT := New("definition") // name will be updated once we know it.
		//				newT.text = t.text
		//				newT.ParseName = t.ParseName
		//				newT.startParse(t.funcs, t.lex, t.treeSet)
		//				newT.parseDefinition()
		//				continue
		//			}
		//	t.backup2(delim)
		//}

		switch n := t.textOrDirective(); n.Type() {
		case nodeEnd, nodeElse:
			t.errorf("unexpected %s", n)
		default:
			t.Root.append(n)
		}
	}
}

// parseDefinition parses a {{define}} ...  {{end}} template definition and
// installs the definition in t.treeSet. The "define" keyword has already
// been scanned.
func (t *Tree) parseDefinition() {
	const context = "define clause"
	name := t.expect(itemStringConstant, context)
	var err error
	t.Name, err = strconv.Unquote(name.val)
	if err != nil {
		t.error(err)
	}
	//	t.expect(itemRightDelim, context)
	var end Node
	t.Root, end = t.itemContent()
	if end.Type() != nodeEnd {
		t.errorf("unexpected %s in %s", end, context)
	}
	t.add()
	t.stopParse()
}

// itemContent:
//	textOrDirective*
// Terminates at </#.
func (t *Tree) itemContent() (content *ContentNode, next Node) {
	content = t.newContent(t.peekNonSpace().pos)

	for t.peekNonSpace().typ != itemEOF {
		n := t.textOrDirective()
		switch n.Type() {
		case nodeEnd, nodeElse:
			return content, n
		}

		content.append(n)
	}

	t.errorf("unexpected EOF")

	return
}

// textOrDirective:
//	text | directive
func (t *Tree) textOrDirective() Node {
	switch token := t.nextNonSpace(); token.typ {
	case itemText:
		return t.newText(token.pos, token.val)
	case itemStartDirective:
		return t.directive()
	case itemEndDirective:
		return t.newEnd(token.pos, token.val)
	default:
		t.unexpected(token, "input")
	}

	return nil
}

// Directive:
//	control
// Left delim "<#" is past. Now get directives.
// First word could be a keyword such as if.
func (t *Tree) directive() (n Node) {
	switch token := t.nextNonSpace(); token.typ {
	case itemDirectiveIf:
		return t.ifControl()
	case itemDirectiveElse:
		return t.elseControl()
	}

	t.backup()
	token := t.peek()

	fmt.Println("directive", token)

	return nil
}

// Expression:
//	node+
func (t *Tree) expression(context string) (expr *ExpressionNode) {
	token := t.peekNonSpace()

	expr = t.newExpression(token.pos)

	operatorStack := &stack{}
	operandStack := &stack{}

	for {
		switch token := t.nextNonSpace(); token.typ {
		case itemRightParen:
			t.backup()
			return
		case itemCloseDirective:
			return
		case itemBool, itemCharConstant, itemStringConstant, itemIdentifier,
			itemNumber, itemLeftParen:

			operandStack.push(&token)
		case itemAdd, itemMinus, itemMultiply, itemDivide, itemLess, itemLessEq:
			top := operatorStack.peek()

			if nil != top && token.precedence() < top.(item).precedence() {
				operandStack.pop()
				operandStack.pop()
			} else {
				operatorStack.push(&token)
			}
		default:
			t.unexpected(token, context)
		}
	}
}

func (t *Tree) parseControl(allowElseIf bool, context string) (pos Pos, expr *ExpressionNode, list, elseList *ContentNode) {
	expr = t.expression(context)
	var next Node
	list, next = t.itemContent()

	fmt.Println(expr.String())

	switch next.Type() {
	case nodeEnd: // done
	case nodeElse:
		if allowElseIf {
			// Special case for "else if". If the "else" is followed immediately by an "if",
			// the elseControl will have left the "if" token pending. Treat
			//	{{if a}}_{{else if b}}_{{end}}
			// as
			//	{{if a}}_{{else}}{{if b}}_{{end}}{{end}}.
			// To do this, parse the if as usual and stop at it {{end}}; the subsequent{{end}}
			// is assumed. This technique works even for long if-else-if chains.
			// TODO: Should we allow else-if in with and range?
			if t.peek().typ == itemDirectiveIf {
				t.next() // Consume the "if" token.
				elseList = t.newContent(next.Position())
				elseList.append(t.ifControl())
				// Do not consume the next item - only one {{end}} required.
				break
			}
		}
		elseList, next = t.itemContent()
		if next.Type() != nodeEnd {
			t.errorf("expected end; found %s", next)
		}
	}

	return expr.Position(), expr, list, elseList
}

// If:
//	<#if expr>itemContent</#if>
//	<#if expr>itemContent<#elseif expr>itemContent<#else>itemContent</#if>
// If keyword is past.
func (t *Tree) ifControl() Node {
	return t.newIf(t.parseControl(true, "if"))
}

// List:
//	<#list expr}}itemContent</#list>
// Range keyword is past.
func (t *Tree) listControl() Node {
	return t.newList(t.parseControl(false, "list"))
}

// Else:
//	{{else}}
// Else keyword is past.
func (t *Tree) elseControl() Node {
	// Special case for "else if".
	peek := t.peekNonSpace()
	if peek.typ == itemDirectiveIf {
		// We see "{{else if ... " but in effect rewrite it to {{else}}{{if ... ".
		return t.newElse(peek.pos)
	}
	token := t.expect(itemCloseDirective, "else")
	return t.newElse(token.pos)
}

func (t *Tree) parseTemplateName(token item, context string) (name string) {
	switch token.typ {
	case itemStringConstant:
		s, err := strconv.Unquote(token.val)
		if err != nil {
			t.error(err)
		}
		name = s
	default:
		t.unexpected(token, context)
	}
	return
}

type stack struct {
	items []interface{}
	count int
}

func (s *stack) push(e interface{}) {
	s.items = append(s.items[:s.count], e)
	s.count++
}

func (s *stack) pop() interface{} {
	if s.count == 0 {
		return nil
	}

	s.count--

	return s.items[s.count]
}

func (s *stack) peek() interface{} {
	if s.count == 0 {
		return nil
	}

	return s.items[s.count]
}
