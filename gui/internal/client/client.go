package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/mauzec/tdsoft/gui/internal/config"
	"go.uber.org/zap"
)

// TODO: add print_dialogs functionality. Without it, we can't use chatid to find chats

type Client struct {
	APIID   string
	APIHash string

	NeedAuth bool

	creatorCmd *exec.Cmd

	UserLogF func(string)
	ExtLog   *zap.Logger

	defaultPyOutHandlers map[string]OutHandler
	defaultPyErrHandlers map[string]ErrHandler

	cfg *config.AppConfig
}

func NewClient(extendedLogger *zap.Logger, appCfg *config.AppConfig) (*Client, error) {
	cl := &Client{}
	var err error

	cl.cfg = appCfg
	if extendedLogger == nil {
		return cl, ErrExtendedLoggerNotProvided
	}
	cl.ExtLog = extendedLogger

	cfg, err := config.LoadConfig[config.TGAPIConfig](appCfg.AuthConfigName, "env", ".")
	if err != nil {
		cl.NeedAuth = true
		_ = cl.DeleteSession()
		return cl, errors.Join(ErrNeedAuth, err)
	}
	cl.APIID = cfg.APIID
	cl.APIHash = cfg.APIHash
	cl.defaultPyOutHandlers = map[string]OutHandler{
		"FLOOD_WAIT": func(t string, pm *PyMsg) {
			cl.ExtLog.Warn("flood wait", zap.Any("details", pm.Details))
			_ = cl.UserLog(2, fmt.Sprintf(
				"Flood wait: %d seconds, program will pause. You can stop it, data has been saved",
				pm.Details["seconds"]),
			)
		},
		"CSV_FLUSH_ERROR": func(t string, pm *PyMsg) {
			cl.ExtLog.Warn("csv flush error", zap.Any("details", pm.Details))
			_ = cl.UserLog(2, "Detected error writing to file. Program wil continue, but data may be broken")
		},
		"ALL_DONE": func(t string, pm *PyMsg) {
			cl.ExtLog.Info("all done", zap.Any("details", pm.Details))
			_ = cl.UserLog(1, "all done, result in "+pm.Details["output"].(string))
		},
		"SCRIPT_STARTED": func(t string, pm *PyMsg) {
			cl.ExtLog.Info("script started", zap.Any("details", pm.Details))
		},
	}
	cl.defaultPyErrHandlers = map[string]ErrHandler{
		"UNCAUGHT_ERROR": func(pm *PyMsg) {
			cl.ExtLog.Error("uncaught error", zap.Any("details", pm.Details))
			_ = cl.UserLog(3, "Something went wrong")
		},
		"TASK_CANCELLED": func(pm *PyMsg) {
			cl.ExtLog.Error("task cancelled", zap.Any("details", pm.Details))
			_ = cl.UserLog(2, "Task cancelled by system")
		},
		"RPC_ERROR": func(pm *PyMsg) {
			cl.ExtLog.Error("rpc error", zap.Any("details", pm.Details))
			_ = cl.UserLog(3, "API error")
		},
		"UNEXPECTED_ERROR": func(pm *PyMsg) {
			cl.ExtLog.Error("unexpected error", zap.Any("details", pm.Details))
			_ = cl.UserLog(3, "Unexpected error occurred")
		},
		"ARGPARSE_ERROR": func(pm *PyMsg) {
			cl.ExtLog.Error("argument parse error", zap.Any("details", pm.Details))
		},
	}

	return cl, nil
}

func (cl *Client) DeleteSession() error {
	if err := os.Remove(cl.cfg.AuthConfigName + ".env"); err != nil {
		return errors.Join(ErrSystemError, err)
	}
	if err := os.Remove(cl.cfg.Session + ".session"); err != nil {
		return errors.Join(ErrSystemError, err)
	}
	return nil
}

func (cl *Client) SaveAPIConfig() error {
	cfg := config.TGAPIConfig{
		APIID:   cl.APIID,
		APIHash: cl.APIHash,
	}
	return config.SaveConfig(&cfg, cl.cfg.AuthConfigName, "env")
}

func (cl *Client) StartCreatorServer() error {

	wait := func(timeout time.Duration) error {
		dl := time.Now().Add(timeout)
		for time.Now().Before(dl) {
			if err := cl.pingCreatorServer(); err == nil {
				return nil // server is good
			}
			time.Sleep(50 * time.Millisecond)
		}
		return ErrCreatorWaitTimeout
	}

	cmd := exec.Command(cl.cfg.VenvPath+"/bin/python3", cl.cfg.ScriptsPath+"/connect.py")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	logf, _ := os.OpenFile(cl.cfg.CreatorLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	cmd.Stdout = logf
	cmd.Stderr = logf
	if err := cmd.Start(); err != nil {
		return err
	}
	cl.creatorCmd = cmd

	err := wait(5 * time.Second)
	if err != nil {
		_ = cl.StopCreatorServer()
		return nil
	}

	cl.ExtLog.Info("creator server started", zap.Int("pid", cmd.Process.Pid))
	return nil
}

func (cl *Client) StopCreatorServer() error {
	if cl.creatorCmd == nil || cl.creatorCmd.Process == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(
		context.Background(), 1*time.Second,
	)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET",
		cl.cfg.CreatorURI+"/shutdown", nil)
	_, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cl.creatorCmd.Wait()
	}()

	pgid := -cl.creatorCmd.Process.Pid
	_ = syscall.Kill(pgid, syscall.SIGINT)

	select {
	case err := <-waitCh:
		cl.creatorCmd = nil
		return err
	case <-time.After(3 * time.Second):
		_ = syscall.Kill(pgid, syscall.SIGKILL)
		err := <-waitCh
		cl.creatorCmd = nil
		if err != nil {
			return err
		}
		return nil
	}
}

func (cl *Client) SendAPIData() error {
	if err := cl.pingCreatorServer(); err != nil {
		return err
	}

	resp, err := http.Get(cl.cfg.CreatorURI + "/api_data?" +
		"api_id=" + cl.APIID + "&api_hash=" + cl.APIHash)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	res := map[string]string{}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return err
	}
	if errMsg, ok := res["error"]; ok {
		return fmt.Errorf("send api data error: %s", errMsg)
	}

	return nil
}

func (cl *Client) SendPhone(phone string) error {
	if err := cl.pingCreatorServer(); err != nil {
		return err
	}

	resp, err := http.Get(cl.cfg.CreatorURI + "/send_code?" +
		"phone=" + phone)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	res := map[string]string{}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return err
	}
	if errMsg, ok := res["error"]; ok {
		return fmt.Errorf("send phone error: %s", errMsg)
	}

	return nil
}

func (cl *Client) SignIn(phone, code string) error {
	if err := cl.pingCreatorServer(); err != nil {
		return err
	}

	resp, err := http.Get(cl.cfg.CreatorURI + "/sign_in?" +
		"phone=" + phone + "&code=" + code)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	res := map[string]string{}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return err
	}
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

	resp, err := http.Get(cl.cfg.CreatorURI + "/check_password?" +
		"password=" + password)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	res := map[string]string{}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return err
	}
	if errMsg, ok := res["error"]; ok {
		return fmt.Errorf("check password error: %s", errMsg)
	}

	return nil
}

func (cl *Client) pingCreatorServer() error {
	resp, err := http.Get(cl.cfg.CreatorURI + "/ping?message=ping")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ErrCreatorPingError
	}

	res := map[string]string{}
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		return ErrCreatorPingError
	}
	if res["message"] != "pong" {
		return ErrCreatorPingError
	}
	return nil
}

func (cl *Client) SetUserLogger(f func(string)) {
	cl.UserLogF = f
}

// 1 - info, 2 - warn, 3 - error
func (cl *Client) UserLog(level int, msg string) error {
	if cl.UserLogF == nil {
		return errors.New("no user log function set")
	}

	switch level {
	case 2:
		cl.UserLogF("Warning! " + msg)
	case 3:
		cl.UserLogF("Error! " + msg)
	default:
		cl.UserLogF(msg)
	}
	return nil
}

// func (cl *Client) SetEmpty() {
// 	cl.APIID = ""
// 	cl.APIHash = ""
// 	cl.NeedAuth = true
// }
