package main

import (
	"errors"
	"os"
	"os/signal"
	"syscall"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"github.com/mauzec/tdsoft/gui/internal/client"
	"github.com/mauzec/tdsoft/gui/internal/config"
	apperrors "github.com/mauzec/tdsoft/gui/internal/errors"
	"github.com/mauzec/tdsoft/gui/internal/ui"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	appCfg, err := config.LoadConfig[config.AppConfig]("app", "toml", "config", ".")
	if err != nil {
		panic("failed to load app config: " + err.Error())
	}

	a := app.NewWithID("tdsoft")
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
			EncodeLevel:  zapcore.CapitalLevelEncoder,
			EncodeCaller: zapcore.ShortCallerEncoder,
			LineEnding:   zapcore.DefaultLineEnding,
		},
		OutputPaths:      []string{"stdout", appCfg.LogPath + "/app.log"},
		ErrorOutputPaths: []string{"stderr", appCfg.LogPath + "/app.log"},
	}
	logger, _ := loggerConfig.Build()

	cl, clientErr := client.NewClient(logger, appCfg, a)
	if clientErr != nil {
		if errors.Is(clientErr, apperrors.ErrNeedAuth) {
			logger.Info("unable to load API config, need auth", zap.Error(clientErr))
		} else {
			logger.Fatal("failed to create client", zap.Error(clientErr))
		}
	}
	r.PutService(a)
	r.PutService(cl)
	r.PutService(w)
	w.SetOnClosed(func() {
		err := cl.StopCreatorServer()
		if err != nil {
			cl.ExtLog.Error("failed to stop creator server", zap.Error(err))
		}

		_ = logger.Sync()
	})

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh

		err := cl.StopCreatorServer()
		if err != nil {
			cl.ExtLog.Error("failed to stop creator server", zap.Error(err))
		}

		_ = logger.Sync()
		os.Exit(0)
	}()

	if clientErr != nil {
		r.Show(ui.ScreenLogin)
	} else {
		r.Show(ui.ScreenMain)
	}

	w.ShowAndRun()
}
