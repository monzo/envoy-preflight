# envoy-preflight

`envoy-preflight` is a simple wrapper application which makes it easier to run applications which depend on envoy as a sidecar container for outside network access. It ensures that your application doesn't start until envoy is ready, and that envoy shuts down when the application exits. It is best used as a prefix to your existing Docker entrypoint. It executes any argument passed to it, doing a simple path lookup:
```
envoy-preflight echo "hi"
envoy-preflight /bin/ls -a
```

The `envoy-preflight` wrapper won't do anything special unless you provide at least the `ENVOY_ADMIN_API` environment variable.  This makes, _e.g._, local development of your app easy.

If you do provide the `ENVOY_ADMIN_API` environment variable, `envoy-preflight`
will poll the proxy indefinitely with backoff, waiting for Envoy to report itself as live.  This implies it has loaded cluster configuration (for example from an ADS server). Only then will it execute the command provided as an argument, so that your app can immediately start accessing the outside network.

All signals are passed to the underlying application. Be warned that `SIGKILL` cannot be passed, so this can leave behind a orphaned process.

When the application exits, as long as it does so with exit code 0, `envoy-preflight` will instruct envoy to shut down immediately.

## Environment variables

| Variable                      | Purpose                                                                                                                                                                                                                                                                                                                                  |
|-------------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `ENVOY_ADMIN_API`             | This is the path to envoy's administration interface, in the format `http://127.0.0.1:9010`. If provided, `envoy-preflight` will poll this url at `/server_info` waiting for envoy to report as `LIVE`. If provided and local (`127.0.0.1` or `localhost`), then envoy will be instructed to shut down if the application exits cleanly. |
| `NEVER_KILL_ISTIO`            | If provided and set to `true`, `envoy-preflight` will not instruct istio to exit under any circumstances.
| `NEVER_KILL_ISTIO_ON_FAILURE` | If provided and set to `true`, `envoy-preflight` will not instruct istio to exit if the main binary has exited with a non-zero exit code.
| `ENVOY_PREFLIGHT_LOGGING`     | If provided and set to `true`, `envoy-preflight` will log various steps to the console which is helpful for debugging |
| `START_WITHOUT_ENVOY`         | If provided and set to `true`, `envoy-preflight` will not wait for envoy to be LIVE before starting the main application. However, it will still instruct envoy to exit.|
| `ISTIO_QUIT_API`              | If provided `envoy-preflight` will send a POST to `/quitquitquit` at the given API.  Should be in format `http://127.0.0.1:15020`.  This is intended for Istio v1.3 and higher.  When not given, Istio will be stopped using a `pkill` command.
| `GENERIC_QUIT_ENDPOINTS`      | If provided `envoy-preflight` will send a POST to the URL given.  Multiple URLs are supported and must be provided as a CSV string.  Should be in format `http://myendpoint.com` or `http://myendpoint.com,https://myotherendpoint.com`.  The status code response is logged (if logging is enabled) but is not used.  A 200 is treated the same as a 404 or 500. `GENERIC_QUIT_ENDPOINTS` is handled before Istio is stopped. |


## How it stops Istio

Envoy-Preflight has two methods to stop Istio.

| Istio Version | Method |
|---------------|--------|
| 1.3 and higher| `/quitquitquit` endpoint |
| 1.2 and lower | `pkill` command

### 1.3 and higher

Version 1.3 of Istio introduced an endpoint `/quitquitquit` similar to Envoy.  By default this endpoint is available at `http://127.0.0.1:15020` which is the Pilot Agent service, responsible for managing envoy. ([Source](https://github.com/istio/istio/issues/15041))

To enable this, set the environment variable `ISTIO_QUIT_API` to `http://127.0.0.1:15020`.

### 1.2 and lower

Versions 1.2 and lower of Istio have no supported method to stop Istio Sidecars.  As a workaround Envoy-Preflight stops Istio using the command `pkill -SIGINT pilot-agent`.

To enable this, you must add `shareProcessNamespace: true` to your **Pod** definition in Kubernetes. This allows Envoy-Preflight to stop the service running on the sidecar container.

*Note:* This method is used by default if `ISTIO_QUIT_API` is not set
