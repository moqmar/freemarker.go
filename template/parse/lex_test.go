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
	"testing"
)

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
	tEOF      = mkItem(itemEOF, "")
	tLinter   = mkItem(itemLeftInterpolation, "${")
	tRinter   = mkItem(itemRightInterpolation, "}")
	tStartDir = mkItem(itemStartDirective, "<#")
	tCloseDir = mkItem(itemCloseDirective, ">")
	tEndDir   = mkItem(itemEndDirective, "</#")

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
		tStartDir,
		tList,
		tSpace,
		mkItem(itemIdentifier, "animals"),
		tSpace,
		tAs,
		tSpace,
		mkItem(itemIdentifier, "animal"),
		tCloseDir,
		tEOF}},
	{"char", `<#if 'a' != 'b'>`, []item{
		tStartDir,
		tIf,
		tSpace,
		mkItem(itemCharConstant, `'a'`),
		tSpace,
		tNeq,
		tSpace,
		mkItem(itemCharConstant, `'b'`),
		tCloseDir,
		tEOF,
	}},
	{"string", `<#if "a" == "b">`, []item{
		tStartDir,
		tIf,
		tSpace,
		mkItem(itemStringConstant, `"a"`),
		tSpace,
		tEq,
		tSpace,
		mkItem(itemStringConstant, `"b"`),
		tCloseDir,
		tEOF,
	}},
	{"bools", "<#if true>", []item{
		tStartDir,
		tIf,
		tSpace,
		mkItem(itemBool, "true"),
		tCloseDir,
		tEOF,
	}},
	{"end if", "<#if true>true content</#if>following content", []item{
		tStartDir,
		tIf,
		tSpace,
		mkItem(itemBool, "true"),
		tCloseDir,
		mkItem(itemText, "true content"),
		tEndDir,
		tIf,
		tCloseDir,
		mkItem(itemText, "following content"),
		tEOF,
	}},
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
