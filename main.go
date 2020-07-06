// Binary scuttle ...
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"time"

	"github.com/cenk/backoff"
	"github.com/monzo/typhon"
)

// ServerInfo ... represents the response from Envoy's server info endpoint
type ServerInfo struct {
	State string `json:"state"`
}

// Version ... Version of the binary, set to value like v1.0.0 in CI using ldflags
var Version = "vlocal"

var (
	config ScuttleConfig
)

func main() {
	config = getConfig()

	log(fmt.Sprintf("Scuttle %s starting up, pid %d", Version, os.Getpid()))

	if len(os.Args) < 2 {
		log("No arguments received, exiting")
		return
	}

	// Check if logging is enabled
	if config.LoggingEnabled {
		log("Logging is now enabled")
	}

	// If an envoy API was set and config is set to wait on envoy
	if config.EnvoyAdminAPI != "" {
		if blockingCtx := waitForEnvoy(); blockingCtx != nil {
			<-blockingCtx.Done()
			err := blockingCtx.Err()
			if err == nil || errors.Is(err, context.Canceled) {
				log("Blocking finished, Envoy has started")
			} else if errors.Is(err, context.DeadlineExceeded) {
				panic(errors.New("timeout reached while waiting for Envoy to start"))
			} else {
				panic(err.Error())
			}
		}
	}

	// Find the executable the user wants to run
	binary, err := exec.LookPath(os.Args[1])
	if err != nil {
		panic(err)
	}

	var proc *os.Process

	// Pass signals to the child process
	// This takes os signal 2 (standard error)
	// and passes those signals to the child process scuttle starts (proc)
	go func() {
		stop := make(chan os.Signal, 2)
		signal.Notify(stop)
		for sig := range stop {
			if proc == nil {
				// Signal received before the process even started. Let's just exit.
				log(fmt.Sprintf("Received signal '%v', exiting", sig))
				kill(1) // Attempt to stop sidecars if configured
			} else {
				// Proc is not null, so the child process is running and should also receive this signal
				log(fmt.Sprintf("Received signal '%v', passing to child", sig))
				proc.Signal(sig)
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

	kill(exitCode)

	os.Exit(exitCode)
}

func kill(exitCode int) {
	switch {
	case config.EnvoyAdminAPI == "":
		// We don't have an ENVOY_ADMIN_API env var, do nothing
		log("kill called, No ENVOY_ADMIN_API, doing nothing")
	case !strings.Contains(config.EnvoyAdminAPI, "127.0.0.1") && !strings.Contains(config.EnvoyAdminAPI, "localhost"):
		// Envoy is not local; do nothing
		log("kill called, ENVOY_ADMIN_API is not localhost or 127.0.0.1, doing nothing")
	case config.NeverKillIstio:
		// We're configured never to kill envoy, do nothing
		log("kill called, NEVER_KILL_ISTIO is true, doing nothing")
	case config.NeverKillIstioOnFailure && exitCode != 0:
		log("kill called, NEVER_KILL_ISTIO_ON_FAILURE is true, exiting without killing Istio")
		os.Exit(exitCode)
	case config.IstioQuitAPI == "":
		// No istio API sent, fallback to Pkill method
		killGenericEndpoints()
		killIstioWithPkill()
	default:
		// Stop istio using api
		killGenericEndpoints()
		killIstioWithAPI()
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
			log(fmt.Sprintf("Sent POST to '%s', error: %s", genericEndpoint, resp.Error))
			continue
		}
		log(fmt.Sprintf("Sent POST to '%s', status code: %d", genericEndpoint, resp.StatusCode))
	}
}

func killIstioWithAPI() {
	log(fmt.Sprintf("Stopping Istio using Istio API '%s' (intended for Istio >v1.2)", config.IstioQuitAPI))

	url := fmt.Sprintf("%s/quitquitquit", config.IstioQuitAPI)
	resp := typhon.NewRequest(context.Background(), "POST", url, nil).Send().Response()
	responseSuccess := false

	if resp.Error != nil {
		log(fmt.Sprintf("Sent quitquitquit to Istio, error: %d", resp.Error))
	} else {
		log(fmt.Sprintf("Sent quitquitquit to Istio, status code: %d", resp.StatusCode))
		responseSuccess = resp.StatusCode == 200
	}

	if !responseSuccess && config.IstioFallbackPkill {
		log(fmt.Sprintf("quitquitquit failed, will attempt pkill method"))
		killIstioWithPkill()
	}
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

func waitForEnvoy() context.Context {
	if config.StartWithoutEnvoy {
		return nil
	}
	var blockingCtx context.Context
	var cancel context.CancelFunc
	if config.WaitForEnvoyTimeout > time.Duration(0) {
		blockingCtx, cancel = context.WithTimeout(context.Background(), config.WaitForEnvoyTimeout)
	} else {
		blockingCtx, cancel = context.WithCancel(context.Background())
	}

	log("Blocking until Envoy starts")
	go pollEnvoy(blockingCtx, cancel)
	return blockingCtx
}

func pollEnvoy(ctx context.Context, cancel context.CancelFunc) {
	url := fmt.Sprintf("%s/server_info", config.EnvoyAdminAPI)

	b := backoff.NewExponentialBackOff()
	// We wait forever for envoy to start. In practice k8s will kill the pod if we take too long.
	b.MaxElapsedTime = config.WaitForEnvoyTimeout

	_ = backoff.Retry(func() error {
		rsp := typhon.NewRequest(ctx, "GET", url, nil).Send().Response()

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
	// Notify the context that it's done, if it has not already been cancelled
	cancel()
}
