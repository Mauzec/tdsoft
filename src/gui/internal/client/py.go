package client

import (
	"bufio"
	"encoding/json"
	"os/exec"
)

type PyMsg struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details"`
}

type PyEnvelope struct {
	Error *PyMsg `json:"error,omitempty"`
	Info  *PyMsg `json:"info,omitempty"`
	Warn  *PyMsg `json:"warn,omitempty"`
	Log   *PyMsg `json:"log,omitempty"`
}

// TODO: need handler for ui to show info/error messages or smth else
func runPyWithStreaming(args []string, onOut func(string, *PyMsg), onErr func(*PyMsg)) error {
	cmd := exec.Command("../../.venv/bin/python3", args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	scOut, scErr := bufio.NewScanner(stdout), bufio.NewScanner(stderr)
	scOut.Buffer(make([]byte, 0, 1024*64), 1<<20)
	scErr.Buffer(make([]byte, 0, 1024*64), 1<<20)

	outDone := make(chan struct{})
	errDone := make(chan struct{})

	go func() {
		defer close(outDone)
		for scOut.Scan() {
			var env PyEnvelope
			if nil == json.Unmarshal(scOut.Bytes(), &env) {
				if onOut == nil {
					continue
				}
				switch {
				case env.Info != nil:
					onOut("INFO", env.Info)
				case env.Warn != nil:
					onOut("WARN", env.Warn)
				case env.Log != nil:
					onOut("LOG", env.Log)
				}
			}
		}
	}()

	go func() {
		defer close(errDone)
		for scErr.Scan() {
			var env PyEnvelope
			if nil == json.Unmarshal(scErr.Bytes(), &env) &&
				env.Error != nil && onErr != nil {
				onErr(env.Error)
				return
			}
		}
	}()

	if err := cmd.Start(); err != nil {
		return err
	}

	waitErr := make(chan error, 1)
	go func() { waitErr <- cmd.Wait() }()
	<-outDone
	<-errDone
	err = <-waitErr

	// if err != nil {
	// 	onErr(&PyMsg{
	// 		Code:    "UNCAUGHT_ERROR",
	// 		Message: "Uncaught error from script",
	// 		Details: map[string]any{
	// 			"error": err.Error(),
	// 		},
	// 	})
	// }

	return nil
}
