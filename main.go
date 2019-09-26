package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"

	"github.com/cenk/backoff"
	"github.com/monzo/typhon"
)

type ServerInfo struct {
	State string `json:"state"`
}

func main() {
	// Should be in format `http://127.0.0.1:9010`
	host, ok := os.LookupEnv("ENVOY_ADMIN_API")
	if ok && os.Getenv("START_WITHOUT_ENVOY") != "true" {
		block(host)
	}

	killAPI, killOk := os.LookupEnv("ENVOY_KILL_API")
	if !killOk {
		killAPI = fmt.Sprintf("%s/quitquitquit", host)
	}

	if len(os.Args) < 2 {
		return
	}

	binary, err := exec.LookPath(os.Args[1])
	if err != nil {
		panic(err)
	}

	var proc *os.Process

	// Pass signals to the child process
	go func() {
		stop := make(chan os.Signal, 2)
		signal.Notify(stop)
		for sig := range stop {
			if proc != nil {
				proc.Signal(sig)
			} else {
				// Signal received before the process even started. Let's just exit.
				os.Exit(1)
			}
		}
	}()

	proc, err = os.StartProcess(binary, os.Args[1:], &os.ProcAttr{
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
	})
	if err != nil {
		panic(err)
	}

	state, err := proc.Wait()
	if err != nil {
		panic(err)
	}

	exitCode := state.ExitCode()

	switch {
	case !ok:
		// We don't have an ENVOY_ADMIN_API env var, do nothing
	case !strings.Contains(host, "127.0.0.1") && !strings.Contains(host, "localhost"):
		// Envoy is not local; do nothing
	case os.Getenv("NEVER_KILL_ENVOY") == "true":
		// We're configured never to kill envoy, do nothing
	case os.Getenv("ALWAYS_KILL_ENVOY") == "true", exitCode == 0:
		// Either we had a clean exit, or we are configured to kill envoy anyway
		_ = typhon.NewRequest(context.Background(), "POST", killAPI, nil).Send().Response()
	}

	os.Exit(exitCode)
}

func block(host string) {
	if os.Getenv("START_WITHOUT_ENVOY") == "true" {
		return
	}

	url := fmt.Sprintf("%s/server_info", host)

	b := backoff.NewExponentialBackOff()
	// We wait forever for envoy to start. In practice k8s will kill the pod if we take too long.
	b.MaxElapsedTime = 0

	_ = backoff.Retry(func() error {
		rsp := typhon.NewRequest(context.Background(), "GET", url, nil).Send().Response()

		info := &ServerInfo{}

		err := rsp.Decode(info)
		if err != nil {
			return err
		}

		if info.State != "LIVE" {
			return errors.New("not live yet")
		}

		return nil
	}, b)
}
