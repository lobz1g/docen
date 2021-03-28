# Docen

Simple utility for generating Dockerfile for golang projects.

## Installation

```shell
go get github.com/lobz1g/docen
```

## Usage

```go
package main

import (
	"log"

	"github.com/lobz1g/docen"
)

func main() {
	err := docen.New().
		SetGoVersion("1.14.9").
		SetPort("3000").
		SetTimezone("Europe/Moscow").
		SetTestMode(true).
		SetAdditionalFolder("my-folder/some-files").
		SetAdditionalFile("another-folder/some-files/file").
		GenerateDockerfile()
	if err != nil {
		log.Fatal(err)
	}
}
```

Dockerfile will be created in the root dir of the project.

```dockerfile
FROM golang:1.14.9-alpine as builder
RUN apk update && apk add --no-cache git ca-certificates tzdata && update-ca-certificates
RUN adduser -D -g '' appuser
RUN mkdir -p /docen
RUN mkdir -p /docen/my-folder/some-files
RUN mkdir -p /docen/another-folder/some-files
COPY . /docen
WORKDIR /docen
RUN CGO_ENABLED=0 go test ./...
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build  -ldflags="-w -s" -o /docen
FROM scratch
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
ENV TZ=Europe/Moscow
COPY --from=builder /docen /docen
COPY --from=builder /docen/my-folder/some-files /docen/my-folder/some-files
COPY --from=builder /docen/another-folder/some-files /docen/another-folder/some-files
COPY --from=builder /docen/another-folder/some-files/file /docen/another-folder/some-files/file
USER appuser
EXPOSE 3000
ENTRYPOINT ["/docen"]
```

## Options

All methods are optional. That means there are default values for success creating Dockerfile without any settings.

Just use below code for generating Dockerfile with default values:

```go
err := docen.New().GenerateDockerfile()
if err != nil {
    log.Fatal(err)
}
```

### Image

By default, the image name is `golang` and the tag is `{your_golang_version}-aplpine`. If the `runtime.Version` method
returns wrong information about your golang version it will be just `alpine` image tag. You can set the version by
method `SetGoVersion`.without any settings.

### Expose port

By default, Dockerfile will be without the expose port field. You can set the port by method `SetPort`. The argument can
be as a single value of port, for example `3000`, as range of values, for example `3000-4000`.

### Timezone

By default, Dockerfile will be without the timezone env field. You can set the timezone by method `SetTimezone`.

### Testing

The image is built without testing, but you can test the app before build. Use the method `SetTestMode` for it.

### Additional folders to image

Such folders as `assets`, `config`, `static` and `templates` are added to the image. You can add additional folders by
method `SetAdditionalFolder`.

### Additional files to image

You can set additional files which need to be added to the image. Use the `SetAdditionalFile` method for it. It also
adds additional folders for these files.
