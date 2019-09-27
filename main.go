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

var loggingEnabled = false

type ServerInfo struct {
	State string `json:"state"`
}

var (
	config ScuttleConfig
)

func main() {
	config = getConfig()

	// Check if logging is enabled
	if config.LoggingEnabled {
		log("Logging is now enabled")
	}

	// If an envoy API was set and config is set to wait on envoy
	if config.EnvoyAdminAPI != "" && config.StartWithoutEnvoy == false {
		log("Blocking until envoy starts")
		block()
	}

	if len(os.Args) < 2 {
		log("No arguments received, exiting")
		return
	}

	// Find the executable the user wants to run
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
				log("Received exit signal, exiting")
				os.Exit(1)
			}
		}
	}()

	// Start process passed in by user
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
	case config.EnvoyAdminAPI == "":
		// We don't have an ENVOY_ADMIN_API env var, do nothing
		log("No ENVOY_ADMIN_API, doing nothing")
	case !strings.Contains(config.EnvoyAdminAPI, "127.0.0.1") && !strings.Contains(config.EnvoyAdminAPI, "localhost"):
		// Envoy is not local; do nothing
		log("ENVOY_ADMIN_API is not localhost or 127.0.0.1, doing nothing")
	case config.NeverKillIstio:
		// We're configured never to kill envoy, do nothing
		log("NEVER_KILL_ISTIO is true, doing nothing")
	case config.IstioQuitAPI == "":
		// We should stop istio, no istio API set.  Use PKILL
		killIstioWithPkill()
	default:
		// Stop istio using api
		killIstioWithAPI()
	}

	os.Exit(exitCode)
}

func killIstioWithAPI() {
	log(fmt.Sprintf("Stopping Istio using Istio API '%s' (intended for Istio >v1.2)", config.IstioQuitAPI))

	url := fmt.Sprintf("%s/quitquitquit", config.IstioQuitAPI)
	resp := typhon.NewRequest(context.Background(), "POST", url, nil).Send().Response()
	log(fmt.Sprintf("Sent quitquitquit to Istio, status code: %d", resp.StatusCode))
	//ToDo: Fallback to pkill if this fails?
}

func killIstioWithPkill() {
	log("Stopping Istio using pkill command (intended for Istio <v1.3)")

	cmd := exec.Command("sh", "-c", "pkill -SIGINT pilot-agent")
	_, err := cmd.Output()
	if err == nil {
		log("Process pilot-agent successfully stopped")
	} else {
		errorMessage := err.Error()
		log("pilot-agent could not be stopped, err: " + errorMessage)
	}
}

func block() {
	if config.StartWithoutEnvoy {
		return
	}

	url := fmt.Sprintf("%s/server_info", config.EnvoyAdminAPI)

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
