package client

import (
	"bufio"
	"encoding/json"
	"os/exec"
	"strings"
)

type PyMsg struct {
	Code    string         `json:"code"`
	Details map[string]any `json:"details"`
}

type PyEnvelope struct {
	Error *PyMsg `json:"error,omitempty"`
	Info  *PyMsg `json:"info,omitempty"`
	Warn  *PyMsg `json:"warn,omitempty"`
	Log   *PyMsg `json:"log,omitempty"`
}

func runPyWithStreaming(venv string, args []string, onOut func(string, *PyMsg), onErr func(*PyMsg)) error {
	cmd := exec.Command(venv+"/bin/python3", args...)

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
	seenStructErr := false
	var stderrLines []string
	const maxStderrLines = 200

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
			if err := json.Unmarshal(scErr.Bytes(), &env); err == nil && env.Error != nil {
				seenStructErr = true
				if onErr != nil {
					onErr(env.Error)
				}
				continue
			}
			stderrLines = append(stderrLines, scErr.Text())
			if len(stderrLines) > maxStderrLines {
				stderrLines = stderrLines[len(stderrLines)-maxStderrLines:]
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

	if err != nil {
		if seenStructErr {
			return nil
		}
		if onErr != nil {
			onErr(&PyMsg{
				Code: "SCRIPT_UNCAUGHT_ERROR",
				Details: map[string]any{
					"error":  err.Error(),
					"stderr": strings.Join(stderrLines, "\n"),
				},
			})
		}
	}
	return err
}
