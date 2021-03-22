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

// Structs to Check on Certificates from certs store
type CertSpec struct {
	path                string              `json:"path"`
	serialNumber        string              `json:"serial_number"`
	subjectAltNames     []map[string]string `json:"subject_alt_names"`
	daysUntilExpiration string              `json:"days_until_expiration"`
	validFrom           string              `json:"valid_from"`
	expirationTime      string              `json:"expiration_time"`
}

type Certificate struct {
	caCert    []CertSpec `json:"ca_cert"`
	certChain []CertSpec `json:"cert_chain"`
}

type Certificates struct {
	Certificates []Certificate `json:"certificates"`
}

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

// Check status of SDS secrets from Istio to make sure SDS bootstrap was successful
// This SDS data is populated at bootstrap Before the actual xDS connections are established so it should be safe to
// just kill the envoy pod qith `quitquitquit` if we are in a bad state here
func checkEnvoyIstioSDS(host string) error {
	url := fmt.Sprintf("%s/certs", host)

	rsp := typhon.NewRequest(context.Background(), "GET", url, nil).Send().Response()

	certs := &Certificates{}

	err := rsp.Decode(certs)
	if err != nil {
		return err
	}

	// Check number of CERTS is > 1, if not hit the KILL Endpoint
	if len(certs.Certificates) == 1 {
		_ = typhon.NewRequest(context.Background(), "POST", fmt.Sprintf("%s/quitquitquit", host), nil).Send().Response()
	}

	return nil
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

		// If Enabled perform the ENVOY ISTIO SDS WARMING Check
		sdsprotection, ok := os.LookupEnv("ENVOY_ISTIO_SDS_WARMING_PROTECTION")
		if ok && sdsprotection == "true" {
			err := checkEnvoyIstioSDS(host)
			if err != nil {
				return err
			}
		}

		return nil
	}, b)
}
