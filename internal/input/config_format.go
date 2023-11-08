package input

type Keyspec string
type Actionspec string
type Modename string

type InputConfig struct {
	Editor       map[Keyspec]Actionspec `yaml:"editor"`
	StringEditor ModedSpec              `yaml:"string-editor"`
}

type ModedSpec struct {
	Normal map[Keyspec]Actionspec `yaml:"normal"`
	Insert map[Keyspec]Actionspec `yaml:"insert"`
}
