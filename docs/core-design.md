# Core design decisions

This file records decisions provided directly by the project owner. The rough
goal under `experiment_files/` is supporting context, not authority over these
decisions.

## Data path

```text
codex-cli <-> local proxy <-> remote proxy <-> provider
```

## Confirmed invariants

1. Deduplication applies only to request data sent from the local proxy to the
   remote proxy. Provider responses return without deduplication.
2. The local proxy performs content chunking and primary hashing. The remote
   may still verify received data for integrity, but the local machine carries
   the main deduplication CPU cost.
3. `provider_url` belongs to local-proxy configuration and is conveyed to the
   remote proxy as tunnel metadata. The remote proxy does not own a fixed
   provider URL.
4. Neither proxy has provider credentials in its configuration. Credentials
   supplied by the client are transported as opaque request metadata and must
   never be logged or placed in the chunk store.
5. Provider HTTP paths and methods are opaque. `provider_url` is a generic URL
   prefix; the incoming raw path and query are appended without interpreting
   versions, endpoint names, or schemas.
6. The current transport scope is ordinary HTTP with streamed SSE responses.
   WebSocket upgrades may be added separately later.

## Not decided yet

- Local-to-remote authentication and encryption.
- Provider-destination validation and SSRF protections.
- Tunnel framing and request lifecycle.
- Chunking algorithm, digest, and chunk sizes.
- Cache storage and eviction behavior.
- Request spooling and resource limits.

These items should be designed with the project owner before implementation.
