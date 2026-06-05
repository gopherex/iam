# web/ — admin panel

The IAM admin SPA. Built to static assets (`web/dist`) and **served by the Go
server** (`cmd/iam`) — it is not deployed separately. Consumes
[`@gopherex/iam-sdk`](../ts/packages/sdk) to talk to the API.

`make build-web` builds it; `make build` embeds the result into the server
binary. Frontend stack is decided later.
