package client

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/mauzec/tdsoftgui/internal/config"
)

type Client struct {
	APIID   string
	APIHash string

	NeedAuth bool

	cmd *exec.Cmd
}

func NewClient() (*Client, error) {
	cl := &Client{}
	cfg, err := config.LoadConfig[config.TGAPIConfig]("tg_auth", "env", ".")
	if err != nil {
		log.Println("no api config; deleting previouse session if exists")
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

// TODO: move host and port to config

func (cl *Client) StartCreatorServer() error {
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "../../.venv/bin/python3", "../connect.py")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	logf, _ := os.OpenFile("creator_server.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	cmd.Stdout = logf
	cmd.Stderr = logf
	if err := cmd.Start(); err != nil {
		return err
	}
	cl.cmd = cmd

	// TODO: move waiting to config or do it dynamic
	time.Sleep(250 * time.Millisecond)
	log.Println("creator server started, pid:", cmd.Process.Pid)
	log.Println("trying ping")
	return cl.pingCreatorServer()
}

func (cl *Client) StopCreatorServer() error {
	if err := cl.pingCreatorServer(); err == nil {
		resp, err := http.Get("http://127.0.0.1:9001/shutdown")
		if err == nil {
			defer resp.Body.Close()
		}
	}

	if cl.cmd == nil || cl.cmd.Process == nil {
		return nil
	}

	if cl.cmd.ProcessState == nil || !cl.cmd.ProcessState.Exited() {
		done := make(chan error, 1)
		go func() { done <- cl.cmd.Wait() }()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			_ = cl.cmd.Process.Kill()
			<-done
		}
	}

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

func (cl *Client) SaveAPIConfig() error {
	cfg := config.TGAPIConfig{
		APIID:   cl.APIID,
		APIHash: cl.APIHash,
	}
	return config.SaveConfig(cfg, "tg_auth", "env")
}

func (cl *Client) SetEmpty() {
	cl.APIID = ""
	cl.APIHash = ""
	cl.NeedAuth = true
}
