# Release notes

Notable changes will be documented here

## 0.9.19

- link-checker-service:
  - JWT authentication
    - allow credentials in CORS
    - configurable `LCS_USEJWTVALIDATION`, `LCS_DISABLEREQUESTLOGGING`
    - logging the configuration before validating it
  - configurable sequence or URL checker plugins, e.g. with and without using a proxy
  - logging `GOMAXPROCS` on `serve`
  - simplest health & liveness check + version endpoints are now unauthenticated and not rate-limited


## 0.9.18

- link-checker-service:
  - Simple authentication via JWT validation

## 0.9.17

- link-checker-service:
  - HTTP BadRequest on empty request
- sample UI:
  - abort button


## 0.9.16

- link-checker-service:
  - use the browser user agent in the last resort get request
  - cache failed DNS resolution to not block the other requests
  - total request timeout increased


## 0.9.15

- link-checker-service:
  - better feedback when reading in config files
  - fixed DNS resolution for URLs with an explicit port

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
