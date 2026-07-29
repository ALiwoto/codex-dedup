# codex-dedup

`codex-dedup` will reduce upload traffic from a constrained local connection by
deduplicating opaque HTTP request-body chunks between a local proxy and a
remote proxy.

```text
codex-cli <-> local proxy <-> remote proxy <-> provider
```

The provider URL is selected by the local proxy and conveyed to the remote
proxy. Provider credentials remain request headers supplied by the client;
neither proxy has a provider-credential configuration field.

## Current state

The local role is a working protocol-unaware HTTP/SSE pass-through proxy.
Remote tunneling and deduplication are not implemented yet.

Copy `config.sample.ini` to `config.ini`, configure the provider URL prefix, and
run the local proxy with:

```powershell
go run . local
```

The incoming method, request path, query, end-to-end headers, and body are
forwarded without provider-specific parsing. `provider_url` acts only as a
prefix:

```text
provider_url: https://provider.example/prefix
incoming:     /anything/here?stream=true
target:       https://provider.example/prefix/anything/here?stream=true
```

This rule applies equally to any other path or ordinary HTTP method. WebSocket
upgrades are not supported yet.

Use `go run . help` for all command-line options.

Owner-confirmed architectural invariants are tracked in
[`docs/core-design.md`](docs/core-design.md).
