package ui_test

import (
	"fmt"

	"github.com/func/func/ui"
)

func ExampleRows() {
	rows := ui.Rows("foo", "bar", "baz")
	fmt.Println(rows)
	// Output:
	// foo
	// bar
	// baz
}

func ExampleRows_skipEmpty() {
	rows := ui.Rows("foo", "", "baz")
	fmt.Println(rows)
	// Output:
	// foo
	// baz
}

func ExampleCols() {
	cols := ui.Cols("foo", "bar", "baz")
	fmt.Println(cols)
	// Output: foo bar baz
}

func ExampleCols_skipEmpty() {
	cols := ui.Cols("foo", "", "baz")
	fmt.Println(cols)
	// Output: foo baz
}
