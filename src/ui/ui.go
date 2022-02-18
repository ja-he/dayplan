package ui

type UIPanel interface {
	Draw(x, y, w, h int)
}

type MainUIPanel interface {
	UIPanel
	Close()
	NeedsSync()
}
