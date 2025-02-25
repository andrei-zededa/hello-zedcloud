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

Currently the web server handles some additional URL paths, apart from serving
any files in the static directory:
  - `/_/version` - See the web server version.

  - `/_/env` - See the environment variables of the web server process, useful
               for example in combination with https://help.zededa.com/hc/en-us/articles/18691668817179-How-to-add-environment-variables-to-edge-applications .

  - `/_/logs` - This is useful to be able to see the logs of all the requests
                received by the web server. The web server also logs on `stdout`.

  - `/_/crash` - Cause the web server process to exit with an error. The request
                 MUST be an HTTP DELETE, with a query param of `areYouSure` set
                 to the `YesIAMSure` value. Additionally the exit code can be
                 specified with the `exitCode` query param (by default `77`).

  - `/_/alloc` - Cause the server to allocate memory. The `size` query param is
                 used to specify how much memory in bytes, suffixes like `KB`,
                 `MB` and `GB` are supported. This can be used for example to
                 allocate memory until the OOM decides to kill the server process,
                 thus simulating an application issue. The server needs to periodically
                 walk the allocated memory to keep it from being freed by the
                 garbage collector. The additional query param `delay` can be used
                 to lower (or increase for a shorter delay) the CPU usage caused
                 by the periodical walking. `ms`, `s` suffixes are supported, the
                 default is `200ms`.

  - `/_/stats` - See some of the Go runtime statistics. This includes CPU time
                 and allocated memory. NOTE: that these values cannot be directly
                 compared with the per-process Linux kernel statistics.

  - `/_/echo`    - Get back a dump of the HTTP request.

  - `/_/upload`  - Accepts a multi-part file upload and saves the uploaded file
                   locally. Not very useful for the file upload itself however
                   it can be used to simulate traffic towards an edge-app instance
                   (similar to if the edge-app instance would do a download).
                   This would also be subject to any bandwidth limit with which
                   the server is started.

Another additional feature is the `-bw-limit` CLI flag. With it the server can
be started with a *rough* bandwidth limit that will be applied globally (so
concurrent requests will all share this bandwidth limit). This can be useful
to simulate slow network connections, for example when this server is used as
a local HTTP datastore to serve images to an EVE-OS instance. The default is
`2GB/s` which should basically mean unlimited in most scenarios.

# The Zedcloud deployment

**TODO**, see `./zedcloud_deployment`.
