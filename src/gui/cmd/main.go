package main

import (
	"os"
	"os/signal"
	"syscall"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"github.com/mauzec/tdsoftgui/internal/client"
	"github.com/mauzec/tdsoftgui/internal/ui"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	a := app.New()
	w := a.NewWindow("tdsoft")
	w.Resize(fyne.NewSize(400, 400))

	r := ui.NewRouter(w)
	ui.RegisterDefaultScreens(r)

	loggerConfig := zap.Config{
		Level:       zap.NewAtomicLevelAt(zapcore.DebugLevel),
		Development: false,
		Encoding:    "console",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:      "T",
			LevelKey:     "L",
			MessageKey:   "M",
			CallerKey:    "C",
			EncodeTime:   zapcore.ISO8601TimeEncoder,
			EncodeLevel:  zapcore.CapitalColorLevelEncoder,
			EncodeCaller: zapcore.ShortCallerEncoder,
			LineEnding:   zapcore.DefaultLineEnding,
		},
		OutputPaths:      []string{"stdout", "tdsoft.log"},
		ErrorOutputPaths: []string{"stderr", "tdsoft.log"},
	}
	logger, _ := loggerConfig.Build()

	cl, err := client.NewClient(logger)
	r.PutService(cl)

	r.PutService(w)

	w.SetOnClosed(func() {
		if cl != nil {
			_ = cl.StopCreatorServer()
		}
		logger.Sync()
	})

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		if cl != nil {
			_ = cl.StopCreatorServer()
		}
		logger.Sync()
		os.Exit(0)
	}()

	if err != nil {
		r.Show(ui.ScreenLogin)
	} else {
		r.Show(ui.ScreenMain)
	}

	w.ShowAndRun()
}
