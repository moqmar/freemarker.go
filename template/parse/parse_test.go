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
	"flag"
	"fmt"
	"testing"
)

func mkItem(typ itemType, text string) item {
	return item{
		typ: typ,
		val: text,
	}
}

func TestStack(t *testing.T) {
	e1 := mkItem(itemNumber, "1")
	e2 := mkItem(itemAdd, "+")
	e3 := mkItem(itemNumber, "2")

	s := &stack{}
	s.push(&e1)
	s.push(&e2)
	s.push(&e3)

	if "2" != s.pop().(*item).val {
		t.Log("unexpected stack item")
	}

	if "+" != s.pop().(*item).val {
		t.Log("unexpected stack item")
	}

	if "1" != s.pop().(*item).val {
		t.Log("unexpected stack item")
	}
}

var debug = flag.Bool("debug", true, "show the errors produced by the main tests")

type parseTest struct {
	name   string
	input  string
	ok     bool
	result string // what the user would see in an error message.
}

const (
	noError  = true
	hasError = false
)

var parseTests = []parseTest{
	{"empty", "", noError,
		``},
	{"comment", "hello-<#--\n\n\n-->-world", noError,
		`"hello-""-world"`},
	{"spaces", " \t\n", noError,
		`" \t\n"`},
	{"text", "some text", noError,
		`"some text"`},
	//	{"emptyDirective", "<#if></#if>", hasError,
	//		``},
	//	{"simple if", "<#if a == b>true content</#if>following content", noError,
	//		`{{if .X}}"true"{{else}}"false"{{end}}`},
}

var builtins = map[string]interface{}{
	"printf": fmt.Sprintf,
}

func testParse(doCopy bool, t *testing.T) {
	textFormat = "%q"
	defer func() { textFormat = "%s" }()
	for _, test := range parseTests {
		tmpl, err := New(test.name).Parse(test.input, make(map[string]*Tree), builtins)
		switch {
		case err == nil && !test.ok:
			t.Errorf("%q: expected error; got none", test.name)
			continue
		case err != nil && test.ok:
			t.Errorf("%q: unexpected error: %v", test.name, err)
			continue
		case err != nil && !test.ok:
			// expected error, got one
			if *debug {
				fmt.Printf("%s: %s\n\t%s\n", test.name, test.input, err)
			}
			continue
		}
		var result string
		if doCopy {
			result = tmpl.Root.Copy().String()
		} else {
			result = tmpl.Root.String()
		}
		if result != test.result {
			t.Errorf("%s=(%q): got\n\t%v\nexpected\n\t%v", test.name, test.input, result, test.result)
		}
	}
}

func TestParse(t *testing.T) {
	testParse(false, t)
}

//// Same as TestParse, but we copy the node first
//func TestParseCopy(t *testing.T) {
//	testParse(true, t)
//}

type isEmptyTest struct {
	name  string
	input string
	empty bool
}

var isEmptyTests = []isEmptyTest{
	{"empty", ``, true},
	{"nonempty", `hello`, false},
	{"spaces only", " \t\n \t\n", true},
	//	{"definition", `{{define "x"}}something{{end}}`, true},
	//	{"definitions and space", "{{define `x`}}something{{end}}\n\n{{define `y`}}something{{end}}\n\n", true},
	{"definitions and text", "{{define `x`}}something{{end}}\nx\n{{define `y`}}something{{end}}\ny\n", false},
	{"definition and action", "{{define `x`}}something{{end}}{{if 3}}foo{{end}}", false},
}

func TestIsEmpty(t *testing.T) {
	if !IsEmptyTree(nil) {
		t.Errorf("nil tree is not empty")
	}
	for _, test := range isEmptyTests {
		tree, err := New("root").Parse(test.input, make(map[string]*Tree), nil)
		if err != nil {
			t.Errorf("%q: unexpected error: %v", test.name, err)
			continue
		}
		if empty := IsEmptyTree(tree.Root); empty != test.empty {
			t.Errorf("%q: expected %t got %t", test.name, test.empty, empty)
		}
	}
}

func TestErrorContextWithTreeCopy(t *testing.T) {
	tree, err := New("root").Parse("{{if true}}{{end}}", make(map[string]*Tree), nil)
	if err != nil {
		t.Fatalf("unexpected tree parse failure: %v", err)
	}
	treeCopy := tree.Copy()
	wantLocation, wantContext := tree.ErrorContext(tree.Root.Nodes[0])
	gotLocation, gotContext := treeCopy.ErrorContext(treeCopy.Root.Nodes[0])
	if wantLocation != gotLocation {
		t.Errorf("wrong error location want %q got %q", wantLocation, gotLocation)
	}
	if wantContext != gotContext {
		t.Errorf("wrong error location want %q got %q", wantContext, gotContext)
	}
}

// All failures, and the result is a string that must appear in the error message.
//var errorTests = []parseTest{
//	// Check line numbers are accurate.
//	{"unclosed1",
//		"line1\n{{",
//		hasError, `unclosed1:2: unexpected unclosed action in command`},
//	{"unclosed2",
//		"line1\n{{define `x`}}line2\n{{",
//		hasError, `unclosed2:3: unexpected unclosed action in command`},
//	// Specific errors.
//	{"function",
//		"{{foo}}",
//		hasError, `function "foo" not defined`},
//	{"comment",
//		"{{/*}}",
//		hasError, `unclosed comment`},
//	{"lparen",
//		"{{.X (1 2 3}}",
//		hasError, `unclosed left paren`},
//	{"rparen",
//		"{{.X 1 2 3)}}",
//		hasError, `unexpected ")"`},
//	{"space",
//		"{{`x`3}}",
//		hasError, `in operand`},
//	{"idchar",
//		"{{a#}}",
//		hasError, `'#'`},
//	{"charconst",
//		"{{'a}}",
//		hasError, `unterminated character constant`},
//	{"stringconst",
//		`{{"a}}`,
//		hasError, `unterminated quoted string`},
//	{"rawstringconst",
//		"{{`a}}",
//		hasError, `unterminated raw quoted string`},
//	{"number",
//		"{{0xi}}",
//		hasError, `number syntax`},
//	{"multidefine",
//		"{{define `a`}}a{{end}}{{define `a`}}b{{end}}",
//		hasError, `multiple definition of template`},
//	{"eof",
//		"{{range .X}}",
//		hasError, `unexpected EOF`},
//	{"variable",
//		// Declare $x so it's defined, to avoid that error, and then check we don't parse a declaration.
//		"{{$x := 23}}{{with $x.y := 3}}{{$x 23}}{{end}}",
//		hasError, `unexpected ":="`},
//	{"multidecl",
//		"{{$a,$b,$c := 23}}",
//		hasError, `too many declarations`},
//	{"undefvar",
//		"{{$a}}",
//		hasError, `undefined variable`},
//	{"wrongdot",
//		"{{true.any}}",
//		hasError, `unexpected . after term`},
//	{"wrongpipeline",
//		"{{12|false}}",
//		hasError, `non executable command in pipeline`},
//	{"emptypipeline",
//		`{{ ( ) }}`,
//		hasError, `missing value for parenthesized pipeline`},
//}

//func TestErrors(t *testing.T) {
//	for _, test := range errorTests {
//		_, err := New(test.name).Parse(test.input, "", "", make(map[string]*Tree))
//		if err == nil {
//			t.Errorf("%q: expected error", test.name)
//			continue
//		}
//		if !strings.Contains(err.Error(), test.result) {
//			t.Errorf("%q: error %q does not contain %q", test.name, err, test.result)
//		}
//	}
//}

//func TestBlock(t *testing.T) {
//	const (
//		input = `a{{block "inner" .}}bar{{.}}baz{{end}}b`
//		outer = `a{{template "inner" .}}b`
//		inner = `bar{{.}}baz`
//	)
//	treeSet := make(map[string]*Tree)
//	tmpl, err := New("outer").Parse(input, "", "", treeSet, nil)
//	if err != nil {
//		t.Fatal(err)
//	}
//	if g, w := tmpl.Root.String(), outer; g != w {
//		t.Errorf("outer template = %q, want %q", g, w)
//	}
//	inTmpl := treeSet["inner"]
//	if inTmpl == nil {
//		t.Fatal("block did not define template")
//	}
//	if g, w := inTmpl.Root.String(), inner; g != w {
//		t.Errorf("inner template = %q, want %q", g, w)
//	}
//}

//func TestLineNum(t *testing.T) {
//	const count = 100
//	text := strings.Repeat("{{printf 1234}}\n", count)
//	tree, err := New("bench").Parse(text, "", "", make(map[string]*Tree), builtins)
//	if err != nil {
//		t.Fatal(err)
//	}
//	// Check the line numbers. Each line is an action containing a template, followed by text.
//	// That's two nodes per line.
//	nodes := tree.Root.Nodes
//	for i := 0; i < len(nodes); i += 2 {
//		line := 1 + i/2
//		// Action first.
//		action := nodes[i].(*ActionNode)
//		if action.Line != line {
//			t.Fatalf("line %d: action is line %d", line, action.Line)
//		}
//		pipe := action.Pipe
//		if pipe.Line != line {
//			t.Fatalf("line %d: pipe is line %d", line, pipe.Line)
//		}
//	}
//}

//func BenchmarkParseLarge(b *testing.B) {
//	text := strings.Repeat("{{1234}}\n", 10000)
//	for i := 0; i < b.N; i++ {
//		_, err := New("bench").Parse(text, "", "", make(map[string]*Tree), builtins)
//		if err != nil {
//			b.Fatal(err)
//		}
//	}
//}
