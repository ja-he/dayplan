package cli

import (
	"fmt"

	"github.com/ja-he/dayplan/internal/intel"
	"github.com/rs/zerolog/log"
)

type DayplanIntelServerRunCommand struct {
	ListenAddr string `short:"l" long:"listen-addr" default:"localhost:8080" description:"Address to listen on"`
	DBPath     string `short:"d" long:"db" default:":memory:" description:"SQLite database path (use ':memory:' for in-memory)"`
}

func (c *DayplanIntelServerRunCommand) Execute(_ []string) error {

	server, err := intel.NewServer(c.ListenAddr, c.DBPath)
	if err != nil {
		return fmt.Errorf("Unable to create server (%w)", err)
	}
	log.Debug().Msgf("Initialized server to listen on %v with DB at %v.", c.ListenAddr, c.DBPath)

	err = server.Run()
	if err != nil {
		return fmt.Errorf("Unable to run server (%w)", err)
	}

	return nil
}
