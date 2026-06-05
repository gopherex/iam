# internal/

Server-private packages. Nothing here may be imported from outside this module
(enforced by Go's `internal/` rule). Public, importable surfaces live in
[`../pkg`](../pkg) instead.

Planned subpackages (added when the stack is fixed): config, server (HTTP
runtime + the ogen server handlers), storage, auth, federation, events,
delivery.
