package tui

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/ja-he/dayplan/src/colors"
	"github.com/ja-he/dayplan/src/model"

	"github.com/gdamore/tcell/v2"
)

type TUIView struct {
	Screen    tcell.Screen
	Model     *TUIModel
	needsSync bool
}

func (v *TUIView) NeedsSync() {
	v.needsSync = true
}

// Initialize the screen checking errors and return it, so long as no critical
// error occurred.
func (t *TUIView) initScreen() {
	s, err := tcell.NewScreen()
	if err != nil {
		log.Fatalf("%+v", err)
	}
	if err := s.Init(); err != nil {
		log.Fatalf("%+v", err)
	}
	t.Screen = s
}

func NewTUIView(tui *TUIModel) *TUIView {
	t := TUIView{}

	t.initScreen()
	defStyle := tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorReset)
	t.Screen.SetStyle(defStyle)
	t.Screen.EnableMouse()
	t.Screen.EnablePaste()
	t.Screen.Clear()

	t.Model = tui
	w, h := t.Screen.Size()
	weather, timeline, tools := 20, 10, 20
	t.Model.UIDim.Initialize(weather, timeline, tools, w, h)

	return &t
}

const editorWidth = 80
const editorHeight = 20

func (t *TUIView) GetScreenCenter() (int, int) {
	w, h := t.Screen.Size()
	x := w / 2
	y := h / 2
	return x, y
}

func (t *TUIView) DrawTools() {
	i := 0

	boxes := t.Model.CalculateCategoryBoxes()
	for _, styling := range t.Model.CategoryStyling.GetAll() {
		box := boxes[styling.Cat]

		t.DrawBox(styling.Style, box.X, box.Y, box.W, box.H)
		t.DrawText(box.X+1, box.Y+1, box.W-2, 0, styling.Style, styling.Cat.Name)
		if t.Model.CurrentCategory == styling.Cat {
			t.DrawBox(colors.DarkenBG(styling.Style, 50), box.X+box.W-1, box.Y, 1, box.H)
		}

		i++
	}
}

func (t *TUIView) DrawEditor() {
	editor := &t.Model.EventEditor
	style := tcell.StyleDefault.Background(tcell.ColorLightGrey).Foreground(tcell.ColorBlack)
	if editor.Active {
		x, y := t.GetScreenCenter()
		x -= editorWidth / 2
		y -= editorHeight / 2
		t.DrawBox(style, x, y, editorWidth, editorHeight)
		t.DrawText(x+1, y+1, editorWidth-2, 0, style, editor.TmpEventInfo.Name)
		t.Screen.ShowCursor(x+1+editor.CursorPos, y+1)
	} else {
		t.Screen.ShowCursor(-1, -1)
	}
}

func (t *TUIView) DrawLog() {
	style := tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack)
	if t.Model.showLog {
		x, y, w, h := 0, 2, t.Model.UIDim.screenWidth, t.Model.UIDim.screenHeight
		t.DrawBox(style, 0, 0, w, h)
		title := "LOG"
		t.DrawBox(style.Background(tcell.ColorLightGrey), 0, 0, w, 1)
		t.DrawText(w/2-len(title)/2, 0, len(title), 1, style.Background(tcell.ColorLightGrey).Bold(true), title)
		for i := len(t.Model.Log.Get()) - 1; i >= 0; i-- {
			entry := &t.Model.Log.Get()[i]

			t.DrawText(x, y, w, 0, style.Foreground(tcell.ColorDarkGrey).Italic(true), entry.Type)
			x += len(entry.Type) + 1

			t.DrawText(x, y, w, 0, style, entry.Message)
			x += len(entry.Message) + 1

			t.DrawText(x, y, w, 0, style.Foreground(tcell.ColorDarkGrey), entry.Location)
			x += len(entry.Location) + 1

			timeStr := strings.Join(strings.Split(entry.At.String(), " ")[0:2], " ")
			t.DrawText(x, y, w, 0, style.Foreground(tcell.ColorLightGrey), timeStr)

			x = 0
			y++
		}
	}
}

func (t *TUIView) Render() {

	t.Screen.Clear()

	t.DrawWeather()
	t.DrawTimeline()
	t.Model.ComputeRects() // TODO: move to controller?
	t.DrawEvents()
	t.DrawTools()
	t.DrawEditor()
	t.DrawLog()
	t.DrawStatus()

	if t.needsSync {
		t.needsSync = false
		t.Screen.Sync()
	} else {
		t.Screen.Show()
	}
}

func (t *TUIView) DrawText(x, y, w, h int, style tcell.Style, text string) {
	row := y
	col := x
	for _, r := range text {
		t.Screen.SetContent(col, row, r, nil, style)
		col++
		if col >= x+w {
			row++
			col = x
		}
		if row > y+h {
			break
		}
	}
}

func (t *TUIView) DrawBox(style tcell.Style, x, y, w, h int) {
	for row := y; row < y+h; row++ {
		for col := x; col < x+w; col++ {
			t.Screen.SetContent(col, row, ' ', nil, style)
		}
	}
}

func (t *TUIView) DrawStatus() {
	screenW, screenH := t.Model.UIDim.ScreenSize()

	x, y := 0, screenH-t.Model.UIDim.StatusHeight()
	w, h := screenW, t.Model.UIDim.StatusHeight()

	statusStyle := tcell.StyleDefault.Background(tcell.ColorLightGray).Foreground(tcell.ColorBlack)
	t.DrawBox(statusStyle, x, y, w, h)
	xs, ys := x, y
	for k, v := range t.Model.Status.Get() {
		statusStr := fmt.Sprintf("%s: %s", k, v)
		t.DrawText(xs, ys, w, h, statusStyle, statusStr)
		xs += len(statusStr)
		if xs >= x+w {
			xs = x
			ys++
			if ys >= y+h {
				break
			}
		}
	}
}

func (t *TUIView) DrawWeather() {
	for timestamp := *model.NewTimestamp("00:00"); timestamp.Legal(); timestamp.Hour++ {
		y := t.Model.toY(timestamp)

		index := model.DayAndTime{
			Date:      t.Model.CurrentDate,
			Timestamp: timestamp,
		}

		weather, ok := t.Model.Weather.Data[index]
		if ok {
			weatherStyle := tcell.StyleDefault.Foreground(tcell.ColorLightBlue)
			switch {
			case weather.PrecipitationProbability > .25:
				weatherStyle = weatherStyle.Background(tcell.NewHexColor(0xccebff)).Foreground(tcell.ColorBlack)
			case weather.Clouds < 25:
				weatherStyle = weatherStyle.Background(tcell.NewHexColor(0xfff0cc)).Foreground(tcell.ColorBlack)
			}

			t.DrawBox(weatherStyle, t.Model.UIDim.WeatherOffset(), y, t.Model.UIDim.WeatherWidth(), t.Model.Resolution)

			t.DrawText(t.Model.UIDim.WeatherOffset(), y, t.Model.UIDim.WeatherWidth(), 0, weatherStyle, weather.Info)
			t.DrawText(t.Model.UIDim.WeatherOffset(), y+1, t.Model.UIDim.WeatherWidth(), 0, weatherStyle, fmt.Sprintf("%2.0f°C", weather.TempC))
			t.DrawText(t.Model.UIDim.WeatherOffset(), y+2, t.Model.UIDim.WeatherWidth(), 0, weatherStyle, fmt.Sprintf("%d%% clouds", weather.Clouds))
			t.DrawText(t.Model.UIDim.WeatherOffset(), y+3, t.Model.UIDim.WeatherWidth(), 0, weatherStyle, fmt.Sprintf("%d%% humidity", weather.Humidity))
			t.DrawText(t.Model.UIDim.WeatherOffset(), y+4, t.Model.UIDim.WeatherWidth(), 0, weatherStyle, fmt.Sprintf("%2.0f%% chance of rain", 100.0*weather.PrecipitationProbability))
		}
	}
}

func (t *TUIView) DrawTimeline() {
	_, height := t.Screen.Size()

	now := time.Now()
	h := now.Hour()
	m := now.Minute()
	if t.Model.Resolution == 0 {
		panic("RES IS ZERO?!")
	}
	nowRow := (h * t.Model.Resolution) - t.Model.ScrollOffset + (m / (60 / t.Model.Resolution))
	cursorRow := t.Model.cursorY

	suntimes := t.Model.Days[t.Model.CurrentDate].SunTimes

	x := t.Model.UIDim.TimelineOffset()
	w := t.Model.UIDim.TimelineWidth()
	for row := 0; row <= height; row++ {
		timestamp := t.Model.TimeAtY(row)

		if timestamp.Hour >= 24 {
			break
		}
		style := tcell.StyleDefault.Foreground(tcell.ColorLightGray)
		if suntimes != nil {
			if !(timestamp.IsAfter(suntimes.Rise)) ||
				(timestamp.IsAfter(suntimes.Set)) {
				style = style.Background(tcell.ColorBlack)
			}
		}
		if row == nowRow && t.Model.CurrentDate.Is(now) {
			t.DrawText(x, row, w, 1,
				style.Foreground(tcell.ColorWhite).Background(tcell.ColorRed).Bold(true),
				fmt.Sprintf("   %s  ", model.NewTimestampFromGotime(now).ToString()))
		} else if row == cursorRow && t.Model.Hovered.EventID != 0 {
			t.DrawText(x, row, w, 1,
				style.Foreground(tcell.ColorLightGrey).Bold(true),
				fmt.Sprintf("   %s  ", t.Model.TimeAtY(cursorRow).ToString()))
		} else if timestamp.Minute == 0 {
			t.DrawText(x, row, w, 1,
				style,
				fmt.Sprintf("   %s  ", timestamp.ToString()))
		} else {
			t.DrawText(x, row, w, 1,
				style,
				"          ")
		}
	}
}

func (t *TUIView) DrawEvents() {
	for _, e := range t.Model.GetCurrentDay().Events {
		style, err := t.Model.CategoryStyling.GetStyle(e.Cat)
		if err != nil {
			t.Model.Log.Add("ERROR", err.Error())
		}
		// based on event state, draw a box or maybe a smaller one, or ...
		p := t.Model.Positions[e.ID]
		if t.Model.Hovered.EventID != e.ID {
			t.DrawBox(style, p.X, p.Y, p.W, p.H)
			t.DrawText(p.X+1, p.Y, p.W-2, p.H, style, e.Name)
			t.DrawText(p.X+p.W-5, p.Y, 5, 1, style, e.Start.ToString())
			t.DrawText(p.X+p.W-5, p.Y+p.H-1, 5, 1, style, e.End.ToString())
		} else {
			selStyle := colors.DarkenBG(style, 80)
			switch t.Model.Hovered.HoverState {
			case HoverStateResize:
				t.DrawBox(style, p.X, p.Y, p.W, p.H-1)
				t.DrawBox(selStyle, p.X, p.Y+p.H-1, p.W, 1)
				t.DrawText(p.X+1, p.Y, p.W-2, p.H, style, e.Name)
				t.DrawText(p.X+p.W-5, p.Y, 5, 1, style, e.Start.ToString())
				t.DrawText(p.X+p.W-5, p.Y+p.H-1, 5, 1, selStyle, e.End.ToString())
			case HoverStateMove:
				t.DrawBox(selStyle, p.X, p.Y, p.W, p.H)
				t.DrawText(p.X+1, p.Y, p.W-2, p.H, selStyle, e.Name)
				t.DrawText(p.X+p.W-5, p.Y, 5, 1, selStyle, e.Start.ToString())
				t.DrawText(p.X+p.W-5, p.Y+p.H-1, 5, 1, selStyle, e.End.ToString())
			case HoverStateEdit:
				t.DrawBox(style, p.X, p.Y, p.W, p.H)
				t.DrawText(p.X+1, p.Y, p.W-2, p.H, selStyle, e.Name)
				t.DrawText(p.X+p.W-5, p.Y, 5, 1, style, e.Start.ToString())
				t.DrawText(p.X+p.W-5, p.Y+p.H-1, 5, 1, style, e.End.ToString())
			default:
				panic("don't know this hover state!")
			}
		}
	}
}
