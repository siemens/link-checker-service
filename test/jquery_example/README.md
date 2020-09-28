## Example UI

- disclaimer: this UI is included for demo purposes only, and does not conscern itself with any security considerations
- run the link checker service with appropriate CORS headers, e.g. `link-checker-service serve  --corsOrigins="http://localhost:8092"`
- serve this directory from the port `8092`, e.g. `python -m http.server 8092`
- open http://localhost:8092
- to create a static executable with the embedded page:
  - run `go generate ./...` to update the embedded server
  - `go build`
  - run the resulting executable
  - to override the UI port, start it with the environment variable `PORT` set. Don't forget to adjust the server's `--corsOrigins` option.
- &rarr; [favicon source](https://favicon.io/favicon-generator/?t=LCS&ff=Actor&fs=77&fc=%23FFFFFF&b=rounded&bc=%23212AF2)
