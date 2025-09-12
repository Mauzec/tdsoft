package ui

import (
	"slices"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

const maxLogLines = 200

type LogGrid struct {
	Grid       *widget.TextGrid
	Scroll     *container.Scroll
	style      widget.TextGridStyle
	flushEvery time.Duration
	maxLines   int

	mu   sync.Mutex
	buf  []string
	stop chan struct{}
}

func NewLogGrid(style widget.TextGridStyle) *LogGrid {
	// TODO: review flushEvery
	grid := widget.NewTextGrid()
	scroll := container.NewVScroll(grid)
	lg := &LogGrid{
		Grid:       grid,
		Scroll:     scroll,
		style:      style,
		maxLines:   maxLogLines,
		buf:        make([]string, 0, maxLogLines+10),
		flushEvery: 123 * time.Millisecond,
		stop:       make(chan struct{}),
	}
	lg.start()
	return lg
}

func (lg *LogGrid) start() {
	t := time.NewTicker(lg.flushEvery)
	go func() {
		defer t.Stop()
		for {
			select {
			case <-t.C:
				lg.flush()
			case <-lg.stop:
				return
			}
		}
	}()
}

// flush transfer buffer to grid rows
func (lg *LogGrid) flush() {
	lg.mu.Lock()
	if len(lg.buf) == 0 {
		lg.mu.Unlock()
		return
	}

	// TODO: optimize memory usage ?
	lines := slices.Clone(lg.buf)
	lg.buf = lg.buf[:0]
	lg.mu.Unlock()

	rows := make([]widget.TextGridRow, 0, len(lines))
	for _, s := range lines {
		cells := make([]widget.TextGridCell, 0, len(s))
		for _, r := range s {
			cells = append(cells, widget.TextGridCell{Rune: r, Style: lg.style})
		}
		rows = append(rows, widget.TextGridRow{Cells: cells})
	}

	fyne.Do(func() {
		b := lg.isAtBottom()
		lg.Grid.Rows = append(lg.Grid.Rows, rows...)
		if len(lg.Grid.Rows) > lg.maxLines {
			lg.Grid.Rows = lg.Grid.Rows[len(lg.Grid.Rows)-lg.maxLines:]
		}
		lg.Grid.Refresh()
		if b {
			lg.Scroll.ScrollToBottom()
		}
	})
}

func (lg *LogGrid) isAtBottom() bool {
	contentHeight := lg.Scroll.Content.Size().Height
	scrollHeight := lg.Scroll.Size().Height

	if contentHeight <= scrollHeight+1 {
		return true
	}

	return lg.Scroll.Offset.Y >=
		(contentHeight-scrollHeight)-float32(2.0)
}

func (lg *LogGrid) Pushback(line string) {
	lg.mu.Lock()
	lg.buf = append(lg.buf, line)
	lg.mu.Unlock()
}

func (lg *LogGrid) Close() {
	close(lg.stop)
}

func (lg *LogGrid) AppendLine(s string) {
	row := widget.TextGridRow{
		Cells: make([]widget.TextGridCell, 0, len(s)),
	}
	for _, r := range s {
		row.Cells = append(row.Cells, widget.TextGridCell{Rune: r})
	}
	lg.Grid.Rows = append(lg.Grid.Rows, row)
	if len(lg.Grid.Rows) > lg.maxLines {
		// TODO: optimize memory for large logs
		lg.Grid.Rows = lg.Grid.Rows[1:]
	}

}

func (lg *LogGrid) Clear() {
	// TODO:
}
