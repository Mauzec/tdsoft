package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/mauzec/tdsoftgui/internal/config"
	"go.uber.org/zap"
)

type Client struct {
	APIID   string
	APIHash string

	NeedAuth bool

	creatorCmd *exec.Cmd

	userLogF func(string)
	extLog   *zap.Logger

	defaultPyOutHandlers map[string]OutHandler
	defaultPyErrHandlers map[string]ErrHandler
}

func NewClient(externalLogger *zap.Logger) (*Client, error) {
	cl := &Client{}
	var err error

	if externalLogger == nil {
		return cl, errors.New("no logger provided")
	}
	cl.extLog = externalLogger

	cl.defaultPyOutHandlers = map[string]OutHandler{
		"FLOOD_WAIT": func(t string, pm *PyMsg) {
			fmt.Printf("[%s] flood wait: %s seconds\n",
				t, pm.Details["seconds"])
		},
		"CSV_FLUSH_ERROR": func(t string, pm *PyMsg) {
			fmt.Printf("[%s] %s\n", t, pm.Message)
		},
	}
	cl.defaultPyErrHandlers = map[string]ErrHandler{
		"UNCAUGHT_ERROR": func(pm *PyMsg) {
			fmt.Printf("[ERROR] uncaught error: %s\n", pm.Details["error"])
		},
		"TASK_CANCELLED": func(pm *PyMsg) {
			fmt.Printf("[ERROR] task cancelled: %s\n", pm.Details["message"])
		},
		"RPC_ERROR": func(pm *PyMsg) {
			fmt.Printf("[ERROR] rpc error: {code: %v, message: %v, id: %v}\n",
				pm.Details["c"], pm.Details["m"], pm.Details["id"])
		},
		"UNEXPECTED_ERROR": func(pm *PyMsg) {
			fmt.Printf("[ERROR] %s:%s\n", pm.Message, pm.Details["error"])
		},
	}

	cfg, err := config.LoadConfig[config.TGAPIConfig]("tg_auth", "env", ".")
	if err != nil {
		cl.extLog.Warn("no api config; deleting previous session if exists", zap.Error(err))

		err = ErrNeedAuth
		cl.NeedAuth = true
		os.Remove("tg_auth.env")
		os.Remove("../test_session.session")
	} else {
		cl.APIID = cfg.APIID
		cl.APIHash = cfg.APIHash
	}

	return cl, err
}

func (cl *Client) DeleteSession() {
	os.Remove("tg_auth.env")
	os.Remove("../test_session.session")
}

func (cl *Client) SaveAPIConfig() error {
	cfg := config.TGAPIConfig{
		APIID:   cl.APIID,
		APIHash: cl.APIHash,
	}
	return config.SaveConfig(cfg, "tg_auth", "env")
}

// TODO: move host and port to config

func (cl *Client) StartCreatorServer() error {
	cmd := exec.Command("../../.venv/bin/python3", "../connect.py")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	logf, _ := os.OpenFile("creator_server.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	cmd.Stdout = logf
	cmd.Stderr = logf
	if err := cmd.Start(); err != nil {
		return err
	}
	cl.creatorCmd = cmd

	// TODO: move waiting to config or do it dynamic
	time.Sleep(250 * time.Millisecond)
	log.Println("creator server started, pid:", cmd.Process.Pid)
	log.Println("trying ping")
	return cl.pingCreatorServer()
}

func (cl *Client) StopCreatorServer() error {
	if cl.creatorCmd == nil {
		return nil
	}
	if err := cl.pingCreatorServer(); err == nil {
		resp, err := http.Get("http://127.0.0.1:9001/shutdown")
		if err == nil {
			defer resp.Body.Close()
		}
	}

	if cl.creatorCmd.Process == nil {
		return nil
	}

	if cl.creatorCmd.ProcessState == nil || !cl.creatorCmd.ProcessState.Exited() {
		done := make(chan error, 1)
		go func() { done <- cl.creatorCmd.Wait() }()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			_ = cl.creatorCmd.Process.Kill()
			<-done
		}
	}

	cl.creatorCmd = nil
	return nil
}

func (cl *Client) SendAPIData() error {
	if err := cl.pingCreatorServer(); err != nil {
		return err
	}

	resp, err := http.Get("http://127.0.0.1:9001/api_data?" +
		"api_id=" + cl.APIID + "&api_hash=" + cl.APIHash)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	res := map[string]string{}
	json.NewDecoder(resp.Body).Decode(&res)
	if errMsg, ok := res["error"]; ok {
		return fmt.Errorf("send api data error: %s", errMsg)
	}

	return nil
}

func (cl *Client) SendPhone(phone string) error {
	if err := cl.pingCreatorServer(); err != nil {
		return err
	}

	resp, err := http.Get("http://127.0.0.1:9001/send_code?" +
		"phone=" + phone)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	res := map[string]string{}
	json.NewDecoder(resp.Body).Decode(&res)
	if errMsg, ok := res["error"]; ok {
		return fmt.Errorf("send phone error: %s", errMsg)
	}

	return nil
}

func (cl *Client) SignIn(phone, code string) error {
	if err := cl.pingCreatorServer(); err != nil {
		return err
	}

	resp, err := http.Get("http://127.0.0.1:9001/sign_in?" +
		"phone=" + phone + "&code=" + code)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	res := map[string]string{}
	json.NewDecoder(resp.Body).Decode(&res)
	if errMsg, ok := res["error"]; ok {
		if strings.HasPrefix(errMsg, "password") {
			return ErrPasswordNeeded
		}
		return fmt.Errorf("sign in error: %s", errMsg)
	}

	return nil
}

func (cl *Client) CheckPassword(password string) error {
	if err := cl.pingCreatorServer(); err != nil {
		return err
	}

	resp, err := http.Get("http://127.0.0.1:9001/check_password?" +
		"password=" + password)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	res := map[string]string{}
	json.NewDecoder(resp.Body).Decode(&res)
	if errMsg, ok := res["error"]; ok {
		return fmt.Errorf("check password error: %s", errMsg)
	}

	return nil
}

func (cl *Client) pingCreatorServer() error {
	resp, err := http.Get("http://127.0.0.1:9001/ping?message=ping")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ErrCreatorPingError
	}

	res := map[string]string{}
	json.NewDecoder(resp.Body).Decode(&res)
	if res["message"] != "pong" {
		return ErrCreatorPingError
	}
	return nil
}

func (cl *Client) SetUserLogger(f func(string)) {
	cl.userLogF = f
}

// 1 - info, 2 - warn, 3 - error
func (cl *Client) UserLog(level int, msg string) error {
	if cl.userLogF == nil {
		return errors.New("no user log function set")
	}

	switch level {
	case 2:
		cl.userLogF("Warning! " + msg)
	case 3:
		cl.userLogF("Error! " + msg)
	default:
		cl.userLogF(msg)
	}
	return nil
}

// func (cl *Client) SetEmpty() {
// 	cl.APIID = ""
// 	cl.APIHash = ""
// 	cl.NeedAuth = true
// }
