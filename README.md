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

| Variable              | Purpose                                                                                                                                                                                                                                                                                                                                  |
|-----------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `ENVOY_ADMIN_API`     | This is the path to envoy's administration interface, in the format `http://127.0.0.1:9010`. If provided, `envoy-preflight` will poll this url at `/server_info` waiting for envoy to report as `LIVE`. If provided and local (`127.0.0.1` or `localhost`), then envoy will be instructed to shut down if the application exits cleanly. |
| `ENVOY_KILL_API`      | This is the endpoint of the POST command to kill envoy, which defaults to `$ENVOY_ADMIN_API/quitquitquit`, but you can provide any value in format `http://127.0.0.1:9010/quitquitquit`. This can be used to support istio by providing the pilot-agent port.                                                                            |
| `NEVER_KILL_ENVOY`    | If provided and set to `true`, `envoy-preflight` will not instruct envoy to exit under any circumstances.                                                                                                                                                                                                                                |
| `ALWAYS_KILL_ENVOY`   | If provided and set to `true`, `envoy-preflight` will instruct envoy to exit, even if the main application exits with a nonzero exit code.                                                                                                                                                                                               |
| `START_WITHOUT_ENVOY` | If provided and set to `true`, `envoy-preflight` will not wait for envoy to be LIVE before starting the main application. However, it will still instruct envoy to exit.                                                                                                                                                                 |
