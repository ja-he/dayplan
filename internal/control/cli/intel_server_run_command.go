package cli

import (
	"fmt"

	"github.com/ja-he/dayplan/internal/intelserver"
)

type DayplanIntelServerRunCommand struct {
	ListenAddr string `short:"l" long:"listen-addr" default:"localhost:8080" description:"Address to listen on"`
}

func (c *DayplanIntelServerRunCommand) Execute(_ []string) error {

	server, err := intelserver.NewServer(c.ListenAddr)
	if err != nil {
		return fmt.Errorf("Unable to create server (%w)", err)
	}
	err = server.Run()
	if err != nil {
		return fmt.Errorf("Unable to run server (%w)", err)
	}

	return nil
}
