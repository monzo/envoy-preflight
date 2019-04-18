# envoy-preflight

`envoy-preflight` is a simple wrapper application which makes it easier to run envoy as a sidecar container. It ensures that the main application doesn't start until envoy is ready, and that envoy shuts down when the application exits. It is best used as a prefix to your existing Docker entrypoint. It executes any argument passed to it, doing a simple path lookup:
```
envoy-preflight echo "hi"
envoy-preflight /bin/ls -a
```

This application, if provided an `ENVOY_ADMIN_API` environment variable, 
will poll indefinitely with backoff, waiting for envoy to report itself as live, implying it has loaded cluster configuration (for example from an ADS server). Only then will it execute the command provided as an argument.

All signals are passed to the underlying application. Be warned that `SIGKILL` cannot be passed, so this can leave behind a orphaned process.

When the application exits, as long as it does so with exit code 0, `envoy-preflight` will instruct envoy to shut down immediately.

## Environment variables

| Variable              | Purpose                                                                                                                                                                                                                                                                                                                                  |
|-----------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `ENVOY_ADMIN_API`     | This is the path to envoy's administration interface, in the format `http://127.0.0.1:9010`. If provided, `envoy-preflight` will poll this url at `/server_info` waiting for envoy to report as `LIVE`. If provided and local (`127.0.0.1` or `localhost`), then envoy will be instructed to shut down if the application exits cleanly. |
| `NEVER_KILL_ENVOY`    | If provided and set to `true`, `envoy-preflight` will not instruct envoy to exit under any circumstances.                                                                                                                                                                                                                                |
| `ALWAYS_KILL_ENVOY`   | If provided and set to `true`, `envoy-preflight` will instruct envoy to exit, even if the main application exits with a nonzero exit code.                                                                                                                                                                                               |
| `START_WITHOUT_ENVOY` | If provided and set to `true`, `envoy-preflight` will not wait for envoy to be LIVE before starting the main application. However, it will still instruct envoy to exit.                                                                                                                                                                 |
