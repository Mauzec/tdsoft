package custom

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
)

type NumericalEntry struct {
	widget.Entry
	AllowFloat bool
}

func NewNumericalEntry() *NumericalEntry {
	e := &NumericalEntry{}
	e.ExtendBaseWidget(e)
	return e
}

func (e *NumericalEntry) TypedRune(r rune) {
	if r >= '0' && r <= '9' {
		e.Entry.TypedRune(r)
		return
	}
	if e.AllowFloat && r == '.' {
		e.Entry.TypedRune(r)
	}
}

func (e *NumericalEntry) isNumber(s string) bool {
	if e.AllowFloat {
		_, err := strconv.ParseFloat(s, 64)
		return err == nil
	}
	_, err := strconv.Atoi(s)
	return err == nil
}

func (e *NumericalEntry) AsFloat() float64 {
	v, _ := strconv.ParseFloat(e.Text, 64)
	return v
}

func (e *NumericalEntry) AsInt() int {
	if e.AllowFloat {
		return int(e.AsFloat())
	}
	v, _ := strconv.Atoi(e.Text)
	return v
}

func (e *NumericalEntry) TypedShortcut(sc fyne.Shortcut) {
	paste, ok := sc.(*fyne.ShortcutPaste)
	if !ok {
		e.Entry.TypedShortcut(sc)
		return
	}

	c := paste.Clipboard.Content()
	if _, err := strconv.ParseFloat(c, 64); err == nil {
		e.Entry.TypedShortcut(sc)
	}
}

func NewNumericalEntryWithData(data binding.String) *NumericalEntry {
	e := NewNumericalEntry()
	e.Bind(data)
	return e
}
