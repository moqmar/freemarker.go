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
	"strings"
	"unicode"
	"unicode/utf8"
)

// Pos represents a byte position in the original input text from which this template was parsed.
type Pos int

// item represents a token or text string returned from the scanner.
type item struct {
	typ  itemType // the type of this item
	pos  Pos      // the starting position, in bytes, of this item in the input string
	val  string   // the value of this item, aka lexeme
	line int      // the line number at the start of this item
}

func (i item) String() string {
	switch {
	case i.typ == itemEOF:
		return "EOF"
	case i.typ == itemError:
		return i.val
	case i.typ > _itemDirective:
		return fmt.Sprintf("<%s>", i.val)
	case len(i.val) > 10:
		return fmt.Sprintf("%.10q...", i.val)
	}

	return fmt.Sprintf("%q", i.val)
}

const (
	LowestPrec  = 0 // non-operators
	UnaryPrec   = 6
	HighestPrec = 7
)

func (i item) precedence() int {
	switch i.typ {
	case itemEq, itemNeq, itemLess, itemLessEq:
		return 3
	case itemAdd, itemMinus:
		return 4
	case itemMultiply, itemDivide:
		return 5
	}

	return LowestPrec
}

// itemType identifies the type of lex items.
type itemType int

// Make the types pretty print.
var itemName = map[itemType]string{
	itemError:          "error",
	itemBool:           "bool",
	itemEOF:            "EOF",
	itemIdentifier:     "identifier",
	itemEq:             "==",
	itemNeq:            "!=",
	itemAdd:            "+",
	itemMinus:          "-",
	itemMultiply:       "*",
	itemDivide:         "/",
	itemLess:           "<",
	itemLessEq:         "<=",
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

const (
	itemError          itemType = iota // error occurred; value is text of error
	itemBool                           // boolean constant
	itemEOF                            // EOF
	itemIdentifier                     // alphanumeric identifier
	itemText                           // plain text
	itemNumber                         // simple number, including imaginary
	itemAdd                            // +
	itemMinus                          // -
	itemMultiply                       // *
	itemDivide                         // /
	itemLess                           // <
	itemLessEq                         // <=
	itemCharConstant                   // character constant
	itemStringConstant                 // string constant
	itemSpace                          // run of spaces separating arguments
	itemEq                             // ==
	itemNeq                            // !=

	itemLeftInterpolation  // ${
	itemRightInterpolation // }
	itemStartDirective     // <#
	itemCloseDirective     // >
	itemEndDirective       // </#
	itemLeftParen          // (
	itemRightParen         // )

	// Directives appear after all the rest.
	_itemDirective       // used only to delimit the directives
	itemDirectiveInclude // include directive
	itemDirectiveMacro   // macro directive
	itemDirectiveIf      // if directive
	itemDirectiveElseif  // elseif directive
	itemDirectiveElse    // else directive
	itemDirectiveList    // list directive
	itemAs               // keyword in list directive

)

var directive = map[string]itemType{
	"include": itemDirectiveInclude,
	"macro":   itemDirectiveMacro,
	"if":      itemDirectiveIf,
	"elseif":  itemDirectiveElseif,
	"else":    itemDirectiveElse,
	"list":    itemDirectiveList,
	"as":      itemAs,
}

const (
	eof        = -1
	spaceChars = " \t\r\n" // These are the space characters defined by Go itself.
)

// stateFn represents the state of the scanner as a function that returns the next state.
type stateFn func(*lexer) stateFn

// lexer holds the state of the scanner.
type lexer struct {
	name       string    // the name of the input; used only for error reports
	input      string    // the string being scanned
	state      stateFn   // the next lexing function to enter
	pos        Pos       // current position in the input
	start      Pos       // start position of this item
	width      Pos       // width of last rune read from input
	lastPos    Pos       // position of most recent item returned by nextItem
	items      chan item // channel of scanned items
	parenDepth int       // nesting depth of ( ) exprs
	line       int       // 1+number of newlines seen
}

// next returns the next rune in the input.
func (l *lexer) next() rune {
	if int(l.pos) >= len(l.input) {
		l.width = 0

		return eof
	}

	r, w := utf8.DecodeRuneInString(l.input[l.pos:])
	l.width = Pos(w)
	l.pos += l.width
	if r == '\n' {
		l.line++
	}

	return r
}

// peek returns but does not consume the next rune in the input.
func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

// backup steps back one rune. Can only be called once per call of next.
func (l *lexer) backup() {
	l.pos -= l.width
	// Correct newline count.
	if l.width == 1 && l.input[l.pos] == '\n' {
		l.line--
	}
}

// emit passes an item back to the client.
func (l *lexer) emit(t itemType) {
	// fmt.Println("emit", l.input[l.start:l.pos])
	l.items <- item{t, l.start, l.input[l.start:l.pos], l.line}

	l.start = l.pos
}

// ignore skips over the pending input before this point.
func (l *lexer) ignore() {
	l.start = l.pos
}

// accept consumes the next rune if it's from the valid set.
func (l *lexer) accept(valid string) bool {
	if strings.ContainsRune(valid, l.next()) {
		return true
	}
	l.backup()
	return false
}

// acceptRun consumes a run of runes from the valid set.
func (l *lexer) acceptRun(valid string) {
	for strings.ContainsRune(valid, l.next()) {
	}
	l.backup()
}

// errorf returns an error token and terminates the scan by passing
// back a nil pointer that will be the next state, terminating l.nextItem.
func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	l.items <- item{itemError, l.start, fmt.Sprintf(format, args...), l.line}
	return nil
}

// nextItem returns the next item from the input.
// Called by the parser, not in the lexing goroutine.
func (l *lexer) nextItem() item {
	item := <-l.items
	l.lastPos = item.pos
	return item
}

// drain drains the output so the lexing goroutine will exit.
// Called by the parser, not in the lexing goroutine.
func (l *lexer) drain() {
	for range l.items {
	}
}

// lex creates a new scanner for the input string.
func lex(name, input string) *lexer {
	l := &lexer{
		name:  name,
		input: input,
		items: make(chan item),
		line:  1,
	}

	go l.run()

	return l
}

// run runs the state machine for the lexer.
func (l *lexer) run() {
	for l.state = lexText; l.state != nil; {
		l.state = l.state(l)
	}

	close(l.items)
}

const (
	leftInterpolation  = "${"
	rightInterpolation = "}"
	leftComment        = "<#--"
	rightComment       = "-->"
	startDirective     = "<#"
	endDirective       = "</#"
	closeDirective     = ">"
)

// State functions.

// lexText scans until an opening interpolation "${", comment "<#--", directive "<#" or "</#".
func lexText(l *lexer) stateFn {
	l.width = 0

	if x := strings.Index(l.input[l.pos:], leftInterpolation); x >= 0 {
		l.pos += Pos(x)
		if l.pos > l.start {
			l.emit(itemText)
		}

		return lexLeftInterpolation
	}

	if x := strings.Index(l.input[l.pos:], leftComment); x >= 0 {
		l.pos += Pos(x)
		if l.pos > l.start {
			l.emit(itemText)
		}

		return lexComment
	}

	if x := strings.Index(l.input[l.pos:], startDirective); x >= 0 {
		l.pos += Pos(x)
		if l.pos > l.start {
			l.emit(itemText)
		}

		return lexDirective
	}

	if x := strings.Index(l.input[l.pos:], endDirective); x >= 0 {
		l.pos += Pos(x)
		if l.pos > l.start {
			l.emit(itemText)
		}

		return lexDirective
	}

	l.pos = Pos(len(l.input))

	// Correctly reached EOF.
	if l.pos > l.start {
		l.emit(itemText)
	}

	l.emit(itemEOF)

	return nil
}

// rightTrimLength returns the length of the spaces at the end of the string.
func rightTrimLength(s string) Pos {
	return Pos(len(s) - len(strings.TrimRight(s, spaceChars)))
}

// atRightInterpolation reports whether the lexer is at a right interpolation "}".
func (l *lexer) atRightInterpolation() bool {
	return strings.HasPrefix(l.input[l.pos:], rightInterpolation)
}

// lexLeftInterpolation scans the left interpolation "${".
func lexLeftInterpolation(l *lexer) stateFn {
	l.pos += Pos(len(leftInterpolation))
	l.emit(itemLeftInterpolation)

	return lexDirective
}

// lexRightInterpolation scans the right interpolation "}", which is known to be present.
func lexRightInterpolation(l *lexer) stateFn {
	l.pos += Pos(len(rightInterpolation))
	l.emit(itemRightInterpolation)

	return lexText
}

// lexComment scans a comment <#-- comment -->.
func lexComment(l *lexer) stateFn {
	l.pos += Pos(len(leftComment))

	i := strings.Index(l.input[l.pos:], rightComment)
	if i < 0 {
		return l.errorf("unclosed comment")
	}

	l.pos += Pos(i + len(rightComment))
	l.ignore() // skip the whole comment text

	return lexText
}

// lexDirective scans the directive inside FTL tags.
func lexDirective(l *lexer) stateFn {
	if strings.HasPrefix(l.input[l.pos:], startDirective) {
		l.pos += Pos(len(startDirective))
		l.emit(itemStartDirective)
	}

	if strings.HasPrefix(l.input[l.pos:], endDirective) {
		l.pos += Pos(len(endDirective))
		l.emit(itemEndDirective)
	}

	r := l.next()
	switch {
	case r == eof || isEndOfLine(r):
		return l.errorf("unclosed directive")
	case isSpace(r):
		return lexSpace
	case r == '+' || r == '-' || ('0' <= r && r <= '9'):
		l.backup()

		return lexNumber
	case r == '"':
		return lexString
	case r == '\'':
		return lexChar
	case r == '!' || r == '=':
		l.backup()

		return lexComparator
	case isAlphaNumeric(r):
		l.backup()

		return lexIdentifier
	case r == '(':
		l.emit(itemLeftParen)
		l.parenDepth++
	case r == ')':
		l.emit(itemRightParen)
		l.parenDepth--

		if l.parenDepth < 0 {
			return l.errorf("unexpected right paren %#U", r)
		}
	//case r <= unicode.MaxASCII && unicode.IsPrint(r):
	//	l.emit(itemCharConstant)
	//
	//	return lexDirective
	case r == '>':
		l.emit(itemCloseDirective)

		return lexText
	case r == '}':
		l.emit(itemRightInterpolation)

		return lexText
	default:
		return l.errorf("unrecognized character in action: %#U", r)
	}

	return lexDirective
}

// lexSpace scans a run of space characters.
// One space has already been seen.
func lexSpace(l *lexer) stateFn {
	for isSpace(l.peek()) {
		l.next()
	}

	l.emit(itemSpace)

	return lexDirective
}

// lexIdentifier scans an alphanumeric.
func lexIdentifier(l *lexer) stateFn {
Loop:
	for {
		switch r := l.next(); {
		case isAlphaNumeric(r):
		// absorb.
		default:
			l.backup()
			word := l.input[l.start:l.pos]
			if !l.atTerminator() {
				return l.errorf("bad character %#U", r)
			}

			switch {
			case directive[word] > _itemDirective:
				l.emit(directive[word])
			case word == "true", word == "false":
				l.emit(itemBool)
			default:
				l.emit(itemIdentifier)
			}

			break Loop
		}
	}

	return lexDirective
}

// lexVariable scans a Variable: $Alphanumeric.
// The $ has been scanned.
func lexVariable(l *lexer) stateFn {
	if l.atTerminator() {
		// Nothing interesting follows -> "$".
		//		l.emit(itemVariable)
		return lexDirective
	}
	//	return lexFieldOrVariable(l, itemVariable)
	return nil
}

// atTerminator reports whether the input is at valid termination character to
// appear after an identifier.
func (l *lexer) atTerminator() bool {
	r := l.peek()
	if isSpace(r) || isEndOfLine(r) {
		return true
	}

	switch r {
	case eof, '.', ',', '|', ':', ')', '(', '>':
		return true
	}
	// Does r start the delimiter? This can be ambiguous (with delim=="//", $x/2 will
	// succeed but should fail) but only in extremely rare cases caused by willfully
	// bad choice of delimiter.
	if rd, _ := utf8.DecodeRuneInString(rightInterpolation); rd == r {
		return true
	}

	return false
}

// lexComparator scans a comparator.
func lexComparator(l *lexer) stateFn {
	comparator := l.next()
	r := l.next()
	if r != '=' {
		return l.errorf("unexpected comparator %#U", r)
	}

	switch string(comparator) + string(r) {
	case "==":
		l.emit(itemEq)
	case "!=":
		l.emit(itemNeq)
	}

	return lexDirective
}

// lexChar scans a character constant.
func lexChar(l *lexer) stateFn {
Loop:
	for {
		switch l.next() {
		case '\\':
			if r := l.next(); r != eof && r != '\n' {
				break
			}
			fallthrough
		case eof, '\n':
			return l.errorf("unterminated character constant")
		case '\'':
			break Loop
		}
	}

	l.emit(itemCharConstant)

	return lexDirective
}

// lexString scans a string constant.
func lexString(l *lexer) stateFn {
Loop:
	for {
		switch l.next() {
		case '\\':
			if r := l.next(); r != eof && r != '\n' {
				break
			}
			fallthrough
		case eof, '\n':
			return l.errorf("unterminated character constant")
		case '"':
			break Loop
		}
	}

	l.emit(itemStringConstant)

	return lexDirective
}

// lexNumber scans a number: decimal, octal, hex, float, or imaginary. This
// isn't a perfect number scanner - for instance it accepts "." and "0x0.2"
// and "089" - but when it's wrong the input is invalid and the parser (via
// strconv) will notice.
func lexNumber(l *lexer) stateFn {
	if !l.scanNumber() {
		return l.errorf("bad number syntax: %q", l.input[l.start:l.pos])
	}
	if sign := l.peek(); sign == '+' || sign == '-' {
		// Complex: 1+2i. No spaces, must end in 'i'.
		if !l.scanNumber() || l.input[l.pos-1] != 'i' {
			return l.errorf("bad number syntax: %q", l.input[l.start:l.pos])
		}
		//		l.emit(itemComplex)
	} else {
		l.emit(itemNumber)
	}
	return lexDirective
}

func (l *lexer) scanNumber() bool {
	// Optional leading sign.
	l.accept("+-")
	// Is it hex?
	digits := "0123456789"
	if l.accept("0") && l.accept("xX") {
		digits = "0123456789abcdefABCDEF"
	}
	l.acceptRun(digits)
	if l.accept(".") {
		l.acceptRun(digits)
	}
	if l.accept("eE") {
		l.accept("+-")
		l.acceptRun("0123456789")
	}
	// Is it imaginary?
	l.accept("i")
	// Next thing mustn't be alphanumeric.
	if isAlphaNumeric(l.peek()) {
		l.next()
		return false
	}
	return true
}

// isSpace reports whether r is a space character.
func isSpace(r rune) bool {
	return r == ' ' || r == '\t'
}

// isEndOfLine reports whether r is an end-of-line character.
func isEndOfLine(r rune) bool {
	return r == '\r' || r == '\n'
}

// isAlphaNumeric reports whether r is an alphabetic, digit or underscore.
func isAlphaNumeric(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}
