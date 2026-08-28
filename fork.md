# Chronosphere fork

This repository is a fork of
[`richardartoul/gobuildcache`](https://github.com/richardartoul/gobuildcache).
The current upstream baseline is v1.5.0.

## Fork changes

| Change | Interface | Upstream status |
| --- | --- | --- |
| Chronosphere module path | `github.com/chronosphereio/gobuildcache` | Fork-only |
| Quiet mode | `-quiet` or `GOBUILDCACHE_QUIET=true` | Fork-only |
| Local fallback | `-local-fallback` or `GOBUILDCACHE_LOCAL_FALLBACK=true` | Upstream contribution planned |

### Chronosphere module path

The module declaration and internal imports use the Chronosphere repository
path. Preserve these paths when synchronizing from upstream.

### Quiet mode

Quiet mode suppresses informational messages from the server and asynchronous
backend while retaining warnings and errors.

### Local fallback

Local fallback applies only when the cache server cannot initialize its remote
backend. It replaces the unavailable backend with the no-op backend, allowing
the server's local disk cache to continue serving Go cache requests. The option
is disabled by default, and cache-clearing commands remain strict.

Chronosphere CI enables this behavior by passing `-local-fallback` in
`GOCACHEPROG`.

## Synchronizing with upstream

When updating the upstream baseline:

1. Merge the selected upstream release into a synchronization branch.
2. Preserve the fork changes listed above while resolving conflicts.
3. Run `go test -short -count=1 ./...` and `go build ./...`.
4. Update the baseline and change inventory in this file.

Remove a change from this inventory when it is adopted upstream and included in
the fork's new baseline.

## CI binary

Chronosphere Buildkite jobs download a prebuilt Linux AMD64 binary from:

```text
gs://chronosphere-go-binaries/gobuildcache-linux-amd64
```

Source changes in this repository do not affect CI until a new binary is built
and uploaded to that object. Before replacing it, retain the current object as
a timestamped backup for rollback.
