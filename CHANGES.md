# Release notes

Notable changes will be documented here

## 0.9.14

- link-checker-service:
  - best effort remote address resolution
- sample UI:
  - show the remote address resolution in the results
  - added a remote_addr column to CSV export

## 0.9.13

- link-checker-service:
  - optional: resolve remote addresses
- sample UI:
  - disable the CSV download button on no check status
  - configurable service URL in the sample UI

## 0.9.12

- new release packaging

## 0.9.11

- link-checker-service:
  - `serve -a <addr>` allows customizing the bind address, e.g. localhost-only: `127.0.0.1:8080`

## 0.9.10

- binaries: link-checker-service, sample UI, sample large list check
- link-checker-service:
  - configurability
  - response-specific caching of results
  - aborting processing on client disconnected
  - rate limiting per domain
  - concurrency limit
  - rate limiting per client
  - optional domain blacklisting using glob patterns
  - Docker deployment
  - URL de-duplication in the request, and duplication in the response
  - optional searching for patterns in HTTP response bodies
  - streaming responses as they arrive
- sample UI:
  - CSV export
  - synchronous and asynchronous requests
  - filtering by status
  - hiding successes
