FROM golang:1.25-bookworm AS build
WORKDIR /src
COPY go.mod ./
COPY embed.go ./
COPY skills ./skills
COPY cmd ./cmd
COPY internal ./internal
RUN go build -ldflags "-X github.com/fluxinc/my-cli/internal/version.Version=0.0.0-container" -o /candidate/my ./cmd/my

FROM ubuntu:24.04
RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends ca-certificates curl git jq util-linux zsh \
    && rm -rf /var/lib/apt/lists/*
COPY --from=build /candidate/my /candidate/my
COPY install.sh /candidate/install.sh
COPY site/public/install.sh /candidate/stable-install.sh
COPY tests/onboarding/scenario.sh /candidate/scenario.sh
RUN chmod +x /candidate/my /candidate/install.sh /candidate/stable-install.sh /candidate/scenario.sh
ENTRYPOINT ["/candidate/scenario.sh"]
