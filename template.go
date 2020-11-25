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

package template

import (
	"reflect"
	"sync"

	"github.com/moqmar/freemarker.go/parse"
)

// missingKeyAction defines how to respond to indexing a map with a key that is not present.
type missingKeyAction int

const (
	mapInvalid   missingKeyAction = iota // Return an invalid reflect.Value.
	mapZeroValue                         // Return the zero value for the map element.
	mapError                             // Error out
)

// common holds the information shared by related templates.
type common struct {
	tmpl map[string]*Template // Map from name to defined templates.
	// We use two maps, one for parsing and one for execution.
	// This separation makes the API cleaner since it doesn't
	// expose reflection to the client.
	muFuncs   sync.RWMutex // protects parseFuncs and execFuncs
	execFuncs map[string]reflect.Value
}

// Template is the representation of a parsed template.
type Template struct {
	name string
	*parse.Tree
	*common
}

// New allocates a new, undefined template with the given name.
func New(name string) *Template {
	t := &Template{
		name: name,
	}
	t.init()
	return t
}

// Name returns the name of the template.
func (t *Template) Name() string {
	return t.name
}

// New allocates a new, undefined template associated with the given one and with the same
// delimiters. The association, which is transitive, allows one template to
// invoke another with a {{template}} action.
func (t *Template) New(name string) *Template {
	t.init()
	nt := &Template{
		name:   name,
		common: t.common,
	}
	return nt
}

// init guarantees that t has a valid common structure.
func (t *Template) init() {
	if t.common == nil {
		c := new(common)
		c.tmpl = make(map[string]*Template)
		c.execFuncs = make(map[string]reflect.Value)
		t.common = c
	}
}

// AddParseTree adds parse tree for template with given name and associates it with t.
// If the template does not already exist, it will create a new one.
// If the template does exist, it will be replaced.
func (t *Template) AddParseTree(name string, tree *parse.Tree) (*Template, error) {
	t.init()
	// If the name is the name of this template, overwrite this template.
	nt := t
	if name != t.name {
		nt = t.New(name)
	}
	// Even if nt == t, we need to install it in the common.tmpl map.
	if replace, err := t.associate(nt, tree); err != nil {
		return nil, err
	} else if replace {
		nt.Tree = tree
	}
	return nt, nil
}

// Templates returns a slice of defined templates associated with t.
func (t *Template) Templates() []*Template {
	if t.common == nil {
		return nil
	}
	// Return a slice so we don't expose the map.
	m := make([]*Template, 0, len(t.tmpl))
	for _, v := range t.tmpl {
		m = append(m, v)
	}
	return m
}

// Lookup returns the template with the given name that is associated with t.
// It returns nil if there is no such template or the template has no definition.
func (t *Template) Lookup(name string) *Template {
	if t.common == nil {
		return nil
	}
	return t.tmpl[name]
}

// Parse parses text as a template body for t.
// Named template definitions ({{define ...}} or {{block ...}} statements) in text
// define additional templates associated with t and are removed from the
// definition of t itself.
//
// Templates can be redefined in successive calls to Parse.
// A template definition with a body containing only white space and comments
// is considered empty and will not replace an existing template's body.
// This allows using Parse to add new named template definitions without
// overwriting the main template body.
func (t *Template) Parse(text string) (*Template, error) {
	t.init()
	t.muFuncs.RLock()
	trees, err := parse.Parse(t.name, text)
	t.muFuncs.RUnlock()
	if err != nil {
		return nil, err
	}
	// Add the newly parsed trees, including the one for t, into our common structure.
	for name, tree := range trees {
		if _, err := t.AddParseTree(name, tree); err != nil {
			return nil, err
		}
	}
	return t, nil
}

// associate installs the new template into the group of templates associated
// with t. The two are already known to share the common structure.
// The boolean return value reports whether to store this tree as t.Tree.
func (t *Template) associate(new *Template, tree *parse.Tree) (bool, error) {
	if new.common != t.common {
		panic("internal error: associate not common")
	}
	if t.tmpl[new.name] != nil && parse.IsEmptyTree(tree.Root) && t.Tree != nil {
		// If a template by that name exists,
		// don't replace it with an empty template.
		return false, nil
	}
	t.tmpl[new.name] = new
	return true, nil
}
