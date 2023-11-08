package edit_test

// import (
// 	"fmt"
// 	"os"
// 	"testing"
// 	"time"
//
// 	"github.com/rs/zerolog"
// 	"github.com/rs/zerolog/log"
//
// 	"github.com/ja-he/dayplan/internal/control/editor"
// 	"github.com/ja-he/dayplan/internal/model"
// )
//
// func TestContstructEditor(t *testing.T) {
// 	log.Logger = log.Output(zerolog.ConsoleWriter{
// 		NoColor:      true,
// 		Out:          os.Stdout,
// 		PartsExclude: []string{"time"},
// 	})
//
// 	task := model.Task{
// 		Name: "Asdfg",
// 		Category: model.Category{
// 			Name:     "Catsanddogs",
// 			Priority: 0,
// 			Goal:     nil,
// 		},
// 		Duration: func() *time.Duration { d := 30 * time.Minute; return &d }(),
// 		Deadline: func() *time.Time {
// 			r, _ := time.Parse("2006-01-02T15:04:05Z07:00", "2006-01-02T15:04:05+07:00")
// 			return &r
// 		}(),
// 		Subtasks: []*model.Task{},
// 	}
// 	e, err := editor.ConstructEditor(&task, nil)
// 	if err != nil {
// 		panic(err)
// 	}
// 	fmt.Println(e)
// }
