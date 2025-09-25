package custom

import (
	"slices"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const maxLogLines = 50

type LogGrid struct {
	Grid       *widget.TextGrid
	Scroll     *container.Scroll
	style      widget.TextGridStyle
	flushEvery time.Duration
	maxLines   int

	mu   sync.Mutex
	buf  []string
	stop chan struct{}

	maxCells int
	contPref []rune
}

func NewLogGrid(style widget.TextGridStyle) *LogGrid {
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

		maxCells: -1,
		contPref: []rune("    "),
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

// flush transfers buffer to grid rows
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

	if lg.maxCells < 0 {
		w := lg.Scroll.Size().Width
		cw := fyne.MeasureText("M", theme.TextSize(),
			fyne.TextStyle{Monospace: true},
		).Width
		lg.maxCells = int(w/cw) + 4
	}

	rows := lg.wrapLines(lines)

	fyne.Do(func() {
		b := lg.isAtBottom()
		lg.Grid.Rows = append(lg.Grid.Rows, rows...)
		if len(lg.Grid.Rows) > lg.maxLines {
			// TODO: optimize memory for large logs
			lg.Grid.Rows = lg.Grid.Rows[len(lg.Grid.Rows)-lg.maxLines:]
		}
		lg.Grid.Refresh()
		if b {
			lg.Scroll.ScrollToBottom()
		}
	})
}

func (lg *LogGrid) wrapLines(lines []string) []widget.TextGridRow {
	firstCols := lg.maxCells
	contCols := firstCols - len(lg.contPref)

	rows := make([]widget.TextGridRow, 0, len(lines))
	for _, s := range lines {
		rns := []rune(s)
		if len(rns) == 0 {
			rows = append(rows, widget.TextGridRow{})
			continue
		}

		// first
		end := min(firstCols, len(rns))
		rows = append(rows, lg.runesToRow(rns[:end]))

		// continuation
		for start := end; start < len(rns); start += contCols {
			currEnd := min(start+contCols, len(rns))

			// prefix n part
			part := make([]rune, 0, len(lg.contPref)+(currEnd-start))
			part = append(part, lg.contPref...)
			part = append(part, rns[start:currEnd]...)
			rows = append(rows, lg.runesToRow(part))
		}
	}
	return rows
}

func (lg *LogGrid) runesToRow(rns []rune) widget.TextGridRow {
	cells := make([]widget.TextGridCell, 0, len(rns))
	for _, r := range rns {
		cells = append(cells, widget.TextGridCell{Rune: r, Style: lg.style})
	}
	return widget.TextGridRow{Cells: cells}
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

func (lg *LogGrid) Clear() {
	lg.mu.Lock()
	lg.buf = lg.buf[:0]
	lg.mu.Unlock()

	fyne.Do(func() {
		// optimize memory?
		lg.Grid.Rows = lg.Grid.Rows[:0]
		lg.Grid.Refresh()
		lg.Scroll.ScrollToTop()
	})
}
