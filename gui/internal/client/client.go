package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"fyne.io/fyne/v2"
	"github.com/mauzec/tdsoft/gui/internal/config"
	apperrors "github.com/mauzec/tdsoft/gui/internal/errors"
	"github.com/mauzec/tdsoft/gui/internal/preferences"
	"go.uber.org/zap"
)

type Client struct {
	APIID    string
	APIHash  string
	NeedAuth bool

	creatorCmd *exec.Cmd

	UserLogF func(string)
	ExtLog   *zap.Logger

	defaultPyOutHandlers map[string]OutHandler
	defaultPyErrHandlers map[string]ErrHandler

	cfg   *config.AppConfig
	prefs fyne.Preferences
}

func NewClient(extendedLogger *zap.Logger, appCfg *config.AppConfig, a fyne.App) (*Client, error) {
	cl := &Client{}
	cl.cfg = appCfg
	if extendedLogger == nil {
		return cl, apperrors.ErrExtendedLoggerNotProvided
	}
	cl.ExtLog = extendedLogger

	if a == nil {
		return cl, errors.New("fyne.App=nil provided")
	}
	cl.prefs = a.Preferences()

	cl.defaultPyOutHandlers = map[string]OutHandler{
		"FLOOD_WAIT": func(t string, pm *PyMsg) {
			cl.ExtLog.Warn("flood wait", zap.Any("details", pm.Details))
			_ = cl.UserLog(2, fmt.Sprintf(
				"Flood wait: %v seconds, program will pause. You can stop it, data has been saved",
				pm.Details["seconds"]))
		},
		"CSV_FLUSH_ERROR": func(t string, pm *PyMsg) {
			cl.ExtLog.Warn("csv flush error", zap.Any("details", pm.Details))
			_ = cl.UserLog(2, "Detected error writing to file. Program wil continue, but data may be broken")
		},
		"ALL_DONE": func(t string, pm *PyMsg) {
			cl.ExtLog.Info("all done", zap.Any("details", pm.Details))
			if out, ok := pm.Details["output"].(string); ok {
				_ = cl.UserLog(1, "All done, result in "+out)
			} else {
				_ = cl.UserLog(1, "All done")
			}
		},
		"SCRIPT_STARTED": func(t string, pm *PyMsg) {
			cl.ExtLog.Info("script started", zap.Any("details", pm.Details))
		},
	}
	cl.defaultPyErrHandlers = map[string]ErrHandler{
		"SCRIPT_UNCAUGHT_ERROR": func(pm *PyMsg) {
			cl.ExtLog.Error("uncaught error", zap.Any("details", pm.Details))
			_ = cl.UserLog(3, "something went wrong")
		},
		"TASK_CANCELLED": func(pm *PyMsg) {
			cl.ExtLog.Error("task cancelled", zap.Any("details", pm.Details))
			_ = cl.UserLog(2, "task cancelled by system")
		},
		"RPC_ERROR": func(pm *PyMsg) {
			cl.ExtLog.Error("rpc error", zap.Any("details", pm.Details))
			_ = cl.UserLog(3, "API error")
		},
		"UNEXPECTED_ERROR": func(pm *PyMsg) {
			cl.ExtLog.Error("unexpected error", zap.Any("details", pm.Details))
			_ = cl.UserLog(3, "unexpected error occurred")
		},
		"ARGPARSE_ERROR": func(pm *PyMsg) {
			cl.ExtLog.Error("argument parse error", zap.Any("details", pm.Details))
		},
		"NO_SESSION": func(pm *PyMsg) {
			cl.ExtLog.Error("no session provided", zap.Any("details", pm.Details))
		},
	}

	APIID := strings.TrimSpace(cl.prefs.String(preferences.KeyTGAPIID))
	APIHash := strings.TrimSpace(cl.prefs.String(preferences.KeyTGAPIHash))
	if _, err := os.Stat(appCfg.Session + ".session"); err != nil ||
		cl.cfg.ForceAuth || APIID == "" || APIHash == "" {
		cl.NeedAuth = true
		_ = os.Remove(cl.cfg.Session + ".session")
		return cl, apperrors.ErrNeedAuth
	}

	return cl, nil
}

func (cl *Client) DeleteSession() error {
	cl.prefs.SetString(preferences.KeyTGAPIID, "")
	cl.prefs.SetString(preferences.KeyTGAPIHash, "")
	cl.prefs.SetString(preferences.KeyTGPhone, "")

	_ = os.Remove(cl.cfg.Session + ".session")
	return nil
}

func (cl *Client) GetDialogs() error {
	// TODO:
	return nil
}

func (cl *Client) CheckConnection() (context.Context, error) {
	// TODO:
	return context.Background(), nil
}

func (cl *Client) SaveAPIConfig() error {
	cl.prefs.SetString(preferences.KeyTGAPIID, strings.TrimSpace(cl.APIID))
	cl.prefs.SetString(preferences.KeyTGAPIHash, strings.TrimSpace(cl.APIHash))
	return nil
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
		return apperrors.ErrCreatorWaitTimeout
	}

	cmd := exec.Command(cl.cfg.VenvPath+"/bin/python3", cl.cfg.ScriptsPath+"/connect.py")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	logf, _ := os.OpenFile(cl.cfg.LogPath+"/creator_server.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	cmd.Stdout = logf
	cmd.Stderr = logf
	if err := cmd.Start(); err != nil {
		return err
	}
	cl.creatorCmd = cmd

	err := wait(5 * time.Second)
	if err != nil {
		_ = cl.StopCreatorServer()
		return err
	}

	err = cl.sendSessionPath()
	if err != nil {
		_ = cl.StopCreatorServer()
		return err
	}
	cl.ExtLog.Info("creator server started", zap.Int("pid", cmd.Process.Pid))

	return nil
}

func (cl *Client) StopCreatorServer() error {
	if cl.creatorCmd == nil || cl.creatorCmd.Process == nil {
		return nil
	}

	// try gracefu http shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", cl.cfg.CreatorURI+"/shutdown", nil)
	if _, err := http.DefaultClient.Do(req); err != nil {
		cl.ExtLog.Warn("creator server shutdown HTTP failed, proceeding to signal", zap.Error(err))
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

func (cl *Client) sendSessionPath() error {
	if err := cl.pingCreatorServer(); err != nil {
		return err
	}

	v := url.Values{}
	v.Set("path", "../"+cl.cfg.Session)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "POST", cl.cfg.CreatorURI+"/session_path?"+v.Encode(), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	res := map[string]string{}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return err
	}
	if errMsg, ok := res["error"]; ok {
		return fmt.Errorf("send session path error: %s", errMsg)
	}

	cl.ExtLog.Info("/session_path response", zap.Any("response", res))

	return nil
}

func (cl *Client) SendAPIData() error {
	if err := cl.pingCreatorServer(); err != nil {
		return err
	}

	v := url.Values{}
	v.Set("api_id", cl.APIID)
	v.Set("api_hash", cl.APIHash)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "POST", cl.cfg.CreatorURI+"/api_data?"+v.Encode(), nil)
	resp, err := http.DefaultClient.Do(req)
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

	cl.ExtLog.Info("/api_data response", zap.Any("response", res))

	return nil
}

func (cl *Client) SendPhone(phone string) error {
	if err := cl.pingCreatorServer(); err != nil {
		return err
	}

	v := url.Values{}
	v.Set("phone", phone)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "POST", cl.cfg.CreatorURI+"/send_code?"+v.Encode(), nil)
	resp, err := http.DefaultClient.Do(req)
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

	cl.ExtLog.Info("/send_code response", zap.Any("response", res))

	return nil
}

func (cl *Client) SignIn(phone, code string) error {
	if err := cl.pingCreatorServer(); err != nil {
		return err
	}

	v := url.Values{}
	v.Set("phone", phone)
	v.Set("code", code)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "POST", cl.cfg.CreatorURI+"/sign_in?"+v.Encode(), nil)
	resp, err := http.DefaultClient.Do(req)
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
			return apperrors.ErrPasswordNeeded
		}
		return fmt.Errorf("sign in error: %s", errMsg)
	}

	cl.ExtLog.Info("/sign_in response", zap.Any("response", res))

	return nil
}

func (cl *Client) CheckPassword(password string) error {
	if err := cl.pingCreatorServer(); err != nil {
		return err
	}

	v := url.Values{}
	v.Set("password", password)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "POST", cl.cfg.CreatorURI+"/check_password?"+v.Encode(), nil)
	resp, err := http.DefaultClient.Do(req)
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

	cl.ExtLog.Info("/check_password response", zap.Any("response", res))

	return nil
}

func (cl *Client) pingCreatorServer() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", cl.cfg.CreatorURI+"/ping?message=ping", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return apperrors.ErrCreatorPingError
	}

	res := map[string]string{}
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		return apperrors.ErrCreatorPingError
	}
	if res["message"] != "pong" {
		return apperrors.ErrCreatorPingError
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

func (cl *Client) ensureUserLogF() error {
	if cl.UserLogF == nil {
		cl.ExtLog.Error("no user log function to set")
		return errors.New("no user log function set")
	}
	return nil
}
