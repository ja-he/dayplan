package main

import (
	"bufio"
	"log"
	"os"

	"dayplan/model"
	"dayplan/termview"
	"dayplan/tui"
)

// MAIN
func main() {
	filename := os.Args[1]
	f, err := os.Open(filename)
	if err != nil {
		log.Fatalf("cannot read file '%s'", filename)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var m model.Model
	for scanner.Scan() {
		s := scanner.Text()
		m.AddEvent(*model.NewEvent(s))
	}

	t := tui.NewTUI()
	t.SetModel(&m)

	tv := termview.NewTermview(t)
	defer tv.Screen.Fini()

	tv.Run()
}
