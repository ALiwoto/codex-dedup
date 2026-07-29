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

The project currently provides the shared command-line, configuration, logging,
and startup foundation. Proxying and deduplication are not implemented yet.

Copy `config.sample.ini` to `config.ini`, fill the local or remote settings, and
validate them with:

```powershell
go run . local --check
go run . remote --check
```

Use `go run . help` for all command-line options.

Owner-confirmed architectural invariants are tracked in
[`docs/core-design.md`](docs/core-design.md).
