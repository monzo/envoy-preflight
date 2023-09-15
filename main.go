package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/cenk/backoff"
	"github.com/monzo/typhon"
)

type ServerInfo struct {
	State string `json:"state"`
}

var (
	config Config
)

func main() {
	log.SetFormatter(&log.JSONFormatter{})
	config = getConfig()

	// Check if logging is enabled
	if config.LoggingEnabled {
		log.Debug("logging is now enabled")
	}

	// If an envoy API was set and config is set to wait on envoy
	if config.EnvoyAdminAPI != "" && config.StartWithoutEnvoy == false {
		log.Info("blocking until envoy starts")
		block()
	}

	if len(os.Args) < 2 {
		log.Info("no arguments received, exiting")
		return
	}

	binary, err := exec.LookPath(os.Args[1])
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
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
				log.Info("received exit signal, exiting")
				os.Exit(1)
			}
		}
	}()

	// Start process passed in by user
	proc, err = os.StartProcess(binary, os.Args[1:], &os.ProcAttr{
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
	})
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}

	state, err := proc.Wait()
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}

	exitCode := state.ExitCode()

	kill(exitCode)

	os.Exit(exitCode)
}

func kill(exitCode int) {
	switch {
	case config.EnvoyAdminAPI == "":
		// We don't have an ENVOY_ADMIN_API env var, do nothing
		log.Info("no ENVOY_ADMIN_API, doing nothing")
	case !strings.Contains(config.EnvoyAdminAPI, "127.0.0.1") && !strings.Contains(config.EnvoyAdminAPI, "localhost"):
		// Envoy is not local; do nothing
		log.Info("ENVOY_ADMIN_API is not localhost or 127.0.0.1, doing nothing")
	case config.NeverKillIstio:
		// We're configured never to kill envoy, do nothing
		log.Info("NEVER_KILL_ISTIO is true, doing nothing")
	case config.NeverKillIstioOnFailure && exitCode != 0:
		log.Info("NEVER_KILL_ISTIO_ON_FAILURE is true, exiting without killing Istio")
	case config.IstioQuitAPI == "":
		// No istio API sent, fallback to Pkill method
		killGenericEndpoints()
		killIstioWithPkill()
	default:
		// Stop istio using api
		killGenericEndpoints()
		status, error := killIstioWithAPI()
		if (error != nil || status != 200) && config.IstioFallbackPkill {
			log.Error("quitquitquit failed, will attempt pkill method")
			killIstioWithPkill()
		}
	}
}

func killGenericEndpoints() {
	if len(config.GenericQuitEndpoints) == 0 {
		return
	}

	for _, genericEndpoint := range config.GenericQuitEndpoints {
		genericEndpoint = strings.Trim(genericEndpoint, " ")
		resp := typhon.NewRequest(context.Background(), "POST", genericEndpoint, nil).Send().Response()
		if resp.Error != nil {
			log.Warnf("sent POST to '%s', error: %s", genericEndpoint, resp.Error)
			continue
		} else {
			log.Infof("sent POST to '%s', status code: %v", genericEndpoint, resp.StatusCode)
		}
	}
}

func killIstioWithAPI() (int, error) {
	log.Infof("stopping  Istio using Istio API '%s' (intended for Istio >v1.2)", config.IstioQuitAPI)

	url := fmt.Sprintf("%s/quitquitquit", config.IstioQuitAPI)
	resp := typhon.NewRequest(context.Background(), "POST", url, nil).Send().Response()
	if resp.Error != nil {
		log.Warnf("sent POST to '%s', error: %s", url, resp.Error)
		return 200, resp.Error
	}
	log.Infof("sent quitquitquit to Istio, status code: %d", resp.StatusCode)

	return resp.StatusCode, resp.Error
}

func killIstioWithPkill() error {
	log.Info("stopping Istio using pkill command (intended for Istio <v1.3)")
	cmd := exec.Command("sh", "-c", "pkill -SIGINT pilot-agent")
	_, err := cmd.Output()
	if err == nil {
		log.Info("process pilot-agent successfully stopped")
	} else {
		errorMessage := err.Error()
		log.Errorf("pilot-agent could not be stopped, err: " + errorMessage)
	}
	return err
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
