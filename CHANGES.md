# Release notes

Notable changes will be documented here

## 0.9.10 preliminary

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
