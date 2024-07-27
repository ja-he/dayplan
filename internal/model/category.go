package model

type CategoryName string

type Category struct {
	Name       CategoryName `dpedit:"name"`
	Priority   int          `dpeditr:"priority"`
	Goal       Goal         `dpedit:",ignore"`
	Deprecated bool         `dpedit:",ignore"`
}

type ByName []Category

func (a ByName) Len() int           { return len(a) }
func (a ByName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByName) Less(i, j int) bool { return a[i].Name < a[j].Name }
