# The application

## hello-zedcloud

`hello-zedcloud` is a small web server that can serve static files (HTML, etc.)
from a directory in the local file system.

It is packaged as a container image that then can be used as a demo / example
application to be deployed through ZEDEDA Cloud to run on an edge-node.

Instead of using a *classic solution* (e.g. `nginx`) `hello-zedcloud` is a small
program written in Go. The reason for this is that we can quickly implement some
additional features in the web server, to demonstrate various features of ZEDEDA
Cloud & of EVE-OS.

## Features

### Static File Serving

The server serves static files (HTML, CSS, JavaScript, images, etc.) from a
configurable directory (default: `./static`).

### Special Endpoints

The web server provides several special endpoints under the `/_/` path:

  - **`/_/version`** (GET) - Returns the web server version.

  - **`/_/env`** (GET) - Returns all environment variables of the web server
    process. Useful for example in combination with https://help.zededa.com/hc/en-us/articles/18691668817179-How-to-add-environment-variables-to-edge-applications.
    *Requires authentication if enabled.*

  - **`/_/logs`** (GET) - Returns all logs of all previous requests received by
    the web server. The web server also logs to `stdout`.
    *Requires authentication if enabled.*

  - **`/_/crash`** (DELETE) - Causes the web server process to exit with an error.
    The request MUST be an HTTP DELETE with a query param `areYouSure=YesIAmSure`.
    Optionally, specify the exit code with `exitCode` query param (default: `77`).
    Example: `curl -X DELETE "http://localhost:10080/_/crash?areYouSure=YesIAmSure&exitCode=42"`
    *Requires authentication if enabled.*

  - **`/_/alloc`** (POST) - Causes the server to allocate memory. The `size` query
    param specifies how much memory in bytes (supports `KB`, `MB`, `GB` suffixes).
    This can be used to allocate memory until the OOM killer terminates the process,
    simulating an application issue. The server periodically walks the allocated
    memory to prevent garbage collection. The optional `delay` query param controls
    CPU usage during memory walking (supports `ms`, `s` suffixes, default: `200ms`).
    Example: `curl -X POST "http://localhost:10080/_/alloc?size=500MB&delay=100ms"`
    *Requires authentication if enabled.*

  - **`/_/stats`** (GET) - Returns Go runtime statistics including CPU time and
    allocated memory. NOTE: These values cannot be directly compared with
    per-process Linux kernel statistics.

  - **`/_/echo`** (ANY) - Returns a complete dump of the HTTP request, including
    headers and body.

  - **`/_/upload`** (POST) - Accepts a multipart file upload and saves it locally.
    Files are stored in `<static-dir>/_/uploads/<upload-id>/` with the original
    filename preserved (after sanitization). The response includes the file path,
    size, and **SHA256 checksum** of the uploaded file. Not very useful for file
    upload itself, but can be used to simulate traffic towards an edge-app instance
    (similar to if the edge-app would download a file). This is subject to any
    bandwidth limit configured for the server.
    Example: `curl -X POST -F "file=@myfile.txt" http://localhost:10080/_/upload`
    *Requires authentication if enabled.*

### HTTP Basic Authentication

The server supports HTTP Basic Authentication for protecting sensitive endpoints.
Authentication can be configured via CLI flags or environment variables:

- By default, random credentials are generated and displayed at startup
- Set `--username=""` (empty string) to disable authentication entirely
- Use `--username=myuser --password=mypass` for custom credentials
- Use `--username=$RANDOM --password=$RANDOM` to generate random credentials

When authentication is enabled, the following endpoints require credentials:
`/_/env`, `/_/logs`, `/_/crash`, `/_/alloc`, `/_/upload`

### Bandwidth Limiting

The server can be started with a global bandwidth limit using the `-bw-limit`
CLI flag. This applies a *rough* bandwidth limit to both read and write operations
(each independently, not combined). Concurrent requests share this bandwidth limit.
This is useful to simulate slow network connections, for example when using this
server as a local HTTP datastore to serve images to an EVE-OS instance.

Supported formats: `2m`, `2mb`, `2M`, `2MB` (all meaning 2 megabytes per second)
Default: `2GB/s` (effectively unlimited in most scenarios)

### Configuration Options

The server can be configured via CLI flags or environment variables:

| CLI Flag | Environment Variable | Default | Description |
|----------|---------------------|---------|-------------|
| `-listen` | `HELLO_LISTEN` | `:10080` | The address (`host:port`) on which the server listens |
| `-static` | `HELLO_STATIC` | `./static` | The directory from which to serve static files |
| `-bw-limit` | `HELLO_BW_LIMIT` | `2GB` | Read and write bandwidth limit (e.g., `2m`, `100MB`) |
| `-username` | `HELLO_USERNAME` | `$RANDOM` | Username for HTTP basic auth (`$RANDOM` = generate random, `""` = disable) |
| `-password` | `HELLO_PASSWORD` | `$RANDOM` | Password for HTTP basic auth (`$RANDOM` = generate random) |

*Note: CLI flags take precedence over environment variables.*

# The Zedcloud deployment

**TODO**, see `./zedcloud_deployment`.
