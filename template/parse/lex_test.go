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
	"fmt"
	"testing"
)

// Make the types prettyprint.
var itemName = map[itemType]string{
	itemError:      "error",
	itemBool:       "bool",
	itemExpression: "expression",
	itemEOF:        "EOF",
	itemIdentifier: "identifier",

	itemCharConstant:   "char",
	itemStringConstant: "string",
	itemNumber:         "number",
	itemLeftParen:      "(",
	itemRightParen:     ")",
	itemSpace:          "space",
	itemText:           "text",

	// directives
	itemDirectiveIf:     "if",
	itemDirectiveElseif: "elseif",
	itemDirectiveElse:   "else",
	itemDirectiveList:   "list",
}

func (i itemType) String() string {
	s := itemName[i]
	if s == "" {
		return fmt.Sprintf("item%d", int(i))
	}
	return s
}

type lexTest struct {
	name  string
	input string
	items []item
}

func mkItem(typ itemType, text string) item {
	return item{
		typ: typ,
		val: text,
	}
}

var (
	tEOF    = mkItem(itemEOF, "")
	tLinter = mkItem(itemLeftInterpolation, "${")
	tRinter = mkItem(itemRightInterpolation, "}")
	tLdir   = mkItem(itemLeftDirective, "<#")
	tRdir   = mkItem(itemRightDirective, ">")

	tLpar    = mkItem(itemLeftParen, "(")
	tRpar    = mkItem(itemRightParen, ")")
	tSpace   = mkItem(itemSpace, " ")
	tInclude = mkItem(itemDirectiveInclude, "include")
	tMacro   = mkItem(itemDirectiveMacro, "macro")
	tIf      = mkItem(itemDirectiveIf, "if")
	tElseif  = mkItem(itemDirectiveElseif, "ellseif")
	tElse    = mkItem(itemDirectiveElse, "else")
	tList    = mkItem(itemDirectiveList, "list")
	tAs      = mkItem(itemAs, "as")
	tEq      = mkItem(itemEq, "==")
	tNeq     = mkItem(itemNeq, "!=")
)

var lexTests = []lexTest{
	{"empty", "", []item{tEOF}},
	{"spaces", " \t\n", []item{mkItem(itemText, " \t\n"), tEOF}},
	{"text", `now is the time`, []item{mkItem(itemText, "now is the time"), tEOF}},
	{"text with comment", "hello-<#-- this is a comment -->-world", []item{
		mkItem(itemText, "hello-"),
		mkItem(itemText, "-world"),
		tEOF,
	}},
	{"empty interpolation", `${}`, []item{tLinter, tRinter, tEOF}},
	{"expression1", "${abc}", []item{
		tLinter,
		mkItem(itemIdentifier, "abc"),
		tRinter,
		tEOF,
	}},
	{"expression2", "${1.5}", []item{
		tLinter,
		mkItem(itemNumber, "1.5"),
		tRinter,
		tEOF,
	}},
	{"list", "<#list animals as animal>", []item{
		tLdir,
		tList,
		tSpace,
		mkItem(itemIdentifier, "animals"),
		tSpace,
		tAs,
		tSpace,
		mkItem(itemIdentifier, "animal"),
		tRdir,
		tEOF}},
	{"char", `<#if 'a' != 'b'>`, []item{
		tLdir,
		tIf,
		tSpace,
		mkItem(itemCharConstant, `'a'`),
		tSpace,
		tNeq,
		tSpace,
		mkItem(itemCharConstant, `'b'`),
		tRdir,
		tEOF,
	}},
	{"string", `<#if "a" == "b">`, []item{
		tLdir,
		tIf,
		tSpace,
		mkItem(itemStringConstant, `"a"`),
		tSpace,
		tEq,
		tSpace,
		mkItem(itemStringConstant, `"b"`),
		tRdir,
		tEOF,
	}},
	{"bools", "<#if true>", []item{
		tLdir,
		tIf,
		tSpace,
		mkItem(itemBool, "true"),
		tRdir,
		tEOF,
	}},
	//	{"variable invocation", "{{$x 23}}", []item{
	//		tLeft,
	//		mkItem(itemVariable, "$x"),
	//		tSpace,
	//		mkItem(itemNumber, "23"),
	//		tRight,
	//		tEOF,
	//	}},

	//	{"declaration", "{{$v := 3}}", []item{
	//		tLeft,
	//		mkItem(itemVariable, "$v"),
	//		tSpace,
	//		mkItem(itemColonEquals, ":="),
	//		tSpace,
	//		mkItem(itemNumber, "3"),
	//		tRight,
	//		tEOF,
	//	}},
	//	{"2 declarations", "{{$v , $w := 3}}", []item{
	//		tLeft,
	//		mkItem(itemVariable, "$v"),
	//		tSpace,
	//		mkItem(itemChar, ","),
	//		tSpace,
	//		mkItem(itemVariable, "$w"),
	//		tSpace,
	//		mkItem(itemColonEquals, ":="),
	//		tSpace,
	//		mkItem(itemNumber, "3"),
	//		tRight,
	//		tEOF,
	//	}},
	//	{"field of parenthesized expression", "{{(.X).Y}}", []item{
	//		tLeft,
	//		tLpar,
	//		mkItem(itemField, ".X"),
	//		tRpar,
	//		mkItem(itemField, ".Y"),
	//		tRight,
	//		tEOF,
	//	}},
	// errors
	//	{"badchar", "#{{\x01}}", []item{
	//		mkItem(itemText, "#"),
	//		tLeft,
	//		mkItem(itemError, "unrecognized character in action: U+0001"),
	//	}},
	//	{"unclosed action", "{{\n}}", []item{
	//		tLeft,
	//		mkItem(itemError, "unclosed action"),
	//	}},
	//	{"EOF in action", "{{range", []item{
	//		tLeft,
	//		tRange,
	//		mkItem(itemError, "unclosed action"),
	//	}},
	//	{"unclosed quote", "{{\"\n\"}}", []item{
	//		tLeft,
	//		mkItem(itemError, "unterminated quoted string"),
	//	}},
	//	{"unclosed raw quote", "{{`xx}}", []item{
	//		tLeft,
	//		mkItem(itemError, "unterminated raw quoted string"),
	//	}},
	//	{"unclosed char constant", "{{'\n}}", []item{
	//		tLeft,
	//		mkItem(itemError, "unterminated character constant"),
	//	}},
	//	{"bad number", "{{3k}}", []item{
	//		tLeft,
	//		mkItem(itemError, `bad number syntax: "3k"`),
	//	}},
	//	{"unclosed paren", "{{(3}}", []item{
	//		tLeft,
	//		tLpar,
	//		mkItem(itemNumber, "3"),
	//		mkItem(itemError, `unclosed left paren`),
	//	}},
	//	{"extra right paren", "{{3)}}", []item{
	//		tLeft,
	//		mkItem(itemNumber, "3"),
	//		tRpar,
	//		mkItem(itemError, `unexpected right paren U+0029 ')'`),
	//	}},

	{"text with bad comment", "hello<#--world", []item{
		mkItem(itemText, "hello"),
		mkItem(itemError, `unclosed comment`),
	}},
}

// collect gathers the emitted items into a slice.
func collect(t *lexTest) (items []item) {
	l := lex(t.name, t.input)
	for {
		item := l.nextItem()
		items = append(items, item)
		if item.typ == itemEOF || item.typ == itemError {
			break
		}
	}
	return
}

func equal(i1, i2 []item, checkPos bool) bool {
	if len(i1) != len(i2) {
		return false
	}
	for k := range i1 {
		if i1[k].typ != i2[k].typ {
			return false
		}
		if i1[k].val != i2[k].val {
			return false
		}
		if checkPos && i1[k].pos != i2[k].pos {
			return false
		}
	}
	return true
}

func TestLex(t *testing.T) {
	for _, test := range lexTests {
		items := collect(&test)
		if !equal(items, test.items, false) {
			t.Errorf("%s: got\n\t%+v\nexpected\n\t%v", test.name, items, test.items)
		}
	}
}

var lexPosTests = []lexTest{
	{"empty", "", []item{tEOF}},
	{"sample", "${abcd}", []item{
		{itemLeftInterpolation, 0, "${", 1},
		{itemIdentifier, 2, "abcd", 1},
		{itemRightInterpolation, 6, "}", 1},
		{itemEOF, 7, "", 1},
	}},
}

// The other tests don't check position, to make the test cases easier to construct.
// This one does.
func TestPos(t *testing.T) {
	for _, test := range lexPosTests {
		items := collect(&test)
		if !equal(items, test.items, true) {
			t.Errorf("%s: got\n\t%v\nexpected\n\t%v", test.name, items, test.items)
			if len(items) == len(test.items) {
				// Detailed print; avoid item.String() to expose the position value.
				for i := range items {
					if !equal(items[i:i+1], test.items[i:i+1], true) {
						i1 := items[i]
						i2 := test.items[i]
						t.Errorf("\t#%d: got {%v %d %q} expected  {%v %d %q}", i, i1.typ, i1.pos, i1.val, i2.typ, i2.pos, i2.val)
					}
				}
			}
		}
	}
}

// TODO D, Test that an error shuts down the lexing goroutine.
//func TestShutdown(t *testing.T) {
//	// We need to duplicate template.Parse here to hold on to the lexer.
//	const text = "erroneous{{define}}{{else}}1234"
//	lexer := lex("foo", text)
//	_, err := New("root").parseLexer(lexer, text)
//	if err == nil {
//		t.Fatalf("expected error")
//	}
//	// The error should have drained the input. Therefore, the lexer should be shut down.
//	token, ok := <-lexer.items
//	if ok {
//		t.Fatalf("input was not drained; got %v", token)
//	}
//}
