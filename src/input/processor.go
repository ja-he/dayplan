package input

type InputProcessor interface {
	HasPartialInput() bool
	ProcessInput(key Key) bool
}
