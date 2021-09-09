# Development

## Running the Tests

```
go test -v ./...
```

## Generating Serializers

```
go generate -v ./...
```

## Load Testing

via [hey](https://github.com/rakyll/hey):

```
hey -m POST -n 10000 -c 300 -T "application/json" -t 30 -D sample_request_body.json http://localhost:8080/checkUrls
```

where the `-c 300` is the client concurrency setting, and `-n 10000` is the approximate total number of requests to fire.

01.09.2020:

```
>hey -m POST -n 10000 -c 200 -T "application/json" -t 30 -D sample_request_body.json http://localhost:8080/checkUrls

Summary:
  Total:        0.2867 secs
  Slowest:      0.0933 secs
  Fastest:      0.0002 secs
  Average:      0.0052 secs
  Requests/sec: 34879.9936

  Total data:   3950000 bytes
  Size/request: 395 bytes

Response time histogram:
  0.000 [1]     |
  0.009 [8720]  |■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
  0.019 [988]   |■■■■■
  0.028 [83]    |
  0.037 [47]    |
  0.047 [57]    |
  0.056 [29]    |
  0.065 [27]    |
  0.075 [15]    |
  0.084 [11]    |
  0.093 [22]    |


Latency distribution:
  10% in 0.0004 secs
  25% in 0.0011 secs
  50% in 0.0032 secs
  75% in 0.0060 secs
  90% in 0.0109 secs
  95% in 0.0146 secs
  99% in 0.0485 secs

Details (average, fastest, slowest):
  DNS+dialup:   0.0004 secs, 0.0002 secs, 0.0933 secs
  DNS-lookup:   0.0004 secs, 0.0000 secs, 0.0262 secs
  req write:    0.0000 secs, 0.0000 secs, 0.0080 secs
  resp wait:    0.0043 secs, 0.0001 secs, 0.0632 secs
  resp read:    0.0003 secs, 0.0000 secs, 0.0117 secs

Status code distribution:
  [200] 10000 responses
```

## Releases

- releases are [automated](github_build.sh) via [.github/workflows/release.yml](.github/workflows/release.yml)) deployment
- locally:
  - assuming a green CI `master` branch
  - update [CHANGES.md](CHANGES.md)
  - test (`go test ./...`)
  - `git tag v<version> -m 'v<version>'`
  - `git push origin v<version>`
  - make sure the release has a comment: `see [CHANGES](CHANGES.md)`
