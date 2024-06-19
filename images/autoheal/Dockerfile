ARG BUILD_IMAGE=golang:1.22

FROM $BUILD_IMAGE as build
ARG TARGETOS
ARG TARGETARCH
ARG LDFLAGS

WORKDIR /workspace
ENV GOPROXY=https://goproxy.cn
COPY go.mod go.sum /workspace/
RUN go mod download
COPY cmd /workspace/cmd
COPY pkg /workspace/pkg
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -a -installsuffix cgo -ldflags "${LDFLAGS}" -o autoheal ./cmd/autoheal/

FROM alpine:3
RUN addgroup -S autoheal && adduser -S autoheal -G autoheal
USER autoheal
EXPOSE 9099

COPY --from=build /workspace/autoheal /usr/local/bin/autoheal
ENTRYPOINT [ "/usr/local/bin/autoheal" ]