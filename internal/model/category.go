package model

import "github.com/lucasb-eyer/go-colorful"

type CategoryName string

type Category struct {
	Name       CategoryName   `dpedit:"name"`
	Priority   int            `dpedit:"priority"`
	Goal       Goal           `dpedit:",ignore"`
	Deprecated bool           `dpedit:",ignore"`
	Color      colorful.Color `dpedit:",ignore"`
}

type ByName []*Category

func (a ByName) Len() int      { return len(a) }
func (a ByName) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByName) Less(i, j int) bool {
	if a[i] == nil || a[j] == nil {
		panic("nil in list of categories")
	}
	return a[i].Name < a[j].Name
}
