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

package template_test

import (
	"log"
	"os"
	"strings"

	"github.com/b3log/freemarker.go/template"
)

func ExampleTTT() {
	const letter = `
It's a simple ${thing}.
`

	// Create a new template and parse the letter into it.
	t := template.Must(template.New("letter").Parse(letter))

	// Execute the template for each recipient.
	for _, r := range recipients {
		err := t.Execute(os.Stdout, r)
		if err != nil {
			log.Println("executing template:", err)
		}
	}

	// Output:111
	// Dear Aunt Mildred,
	//
	// It was a pleasure to see you at the wedding.
	// Thank you for the lovely bone china tea set.
	//
	// Best wishes,
	// Josie
	//
	// Dear Uncle John,
	//
	// It is a shame you couldn't make it to the wedding.
	// Thank you for the lovely moleskin pants.
	//
	// Best wishes,
	// Josie
	//
	// Dear Cousin Rodney,
	//
	// It is a shame you couldn't make it to the wedding.
	//
	// Best wishes,
	// Josie
}
