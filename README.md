# Scuttle

`scuttle` Is a wrapper application that makes it easy to run containers next to Istio sidecars.  It ensures the main application doesn't start until envoy is ready, and that the istio sidecar shuts down when the application exists.  This particularly useful for Jobs that need Istio sidecar injection, as the Istio pod would otherwise run indefinitely after the job is completed.

This application, if provided an `ENVOY_ADMIN_API` environment variable, 
will poll indefinitely with backoff, waiting for envoy to report itself as live, implying it has loaded cluster configuration (for example from an ADS server). Only then will it execute the command provided as an argument.

All signals are passed to the underlying application. Be warned that `SIGKILL` cannot be passed, so this can leave behind a orphaned process.

When the application exits, as long as it does so with exit code 0, `scuttle` will instruct envoy to shut down immediately.

## Environment variables

| Variable              | Purpose                                                                                                                                                                                                                                                                                                                                  |
|-----------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `ENVOY_ADMIN_API`     | This is the path to envoy's administration interface, in the format `http://127.0.0.1:9010`. If provided, `scuttle` will poll this url at `/server_info` waiting for envoy to report as `LIVE`. If provided and local (`127.0.0.1` or `localhost`), then envoy will be instructed to shut down if the application exits cleanly. |
| `NEVER_KILL_ENVOY`    | If provided and set to `true`, `scuttle` will not instruct envoy to exit under any circumstances.    
| `SCUTTLE_LOGGING`    | If provided and set to `true`, `scuttle` will log various steps to the console which is helpful for debugging                                                                                                                                                                                                                                                                                                                                                                     |
| `START_WITHOUT_ENVOY` | If provided and set to `true`, `scuttle` will not wait for envoy to be LIVE before starting the main application. However, it will still instruct envoy to exit.                                                                                                                                                                 |

## Example Dockerfile

```dockerfile
FROM base AS final
WORKDIR /my-app
COPY ["my-app/", "."]
ENTRYPOINT ["scuttle", "dotnet", "test"]
```

## Credits

Origin code is forked from the [envoy-preflight](https://github.com/monzo/envoy-preflight) project on Github, which works for envoy but not for Istio sidecars.
