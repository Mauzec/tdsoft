package main

import (
	"os"
	"os/signal"
	"syscall"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"github.com/mauzec/tdsoftgui/internal/client"
	"github.com/mauzec/tdsoftgui/internal/ui"
)

func main() {
	a := app.New()
	w := a.NewWindow("tdsoft")
	w.Resize(fyne.NewSize(400, 400))

	r := ui.NewRouter(w)
	ui.RegisterDefaultScreens(r)

	cl, err := client.NewClient()
	r.PutService(cl)

	w.SetOnClosed(func() {
		if cl != nil {
			_ = cl.StopCreatorServer()
		}
	})

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		if cl != nil {
			_ = cl.StopCreatorServer()
		}
		os.Exit(0)
	}()

	if err != nil {
		r.Show(ui.ScreenLogin)
	} else {
		r.ShowWith(ui.ScreenTODO, "Main Screen")
	}

	w.ShowAndRun()
}
