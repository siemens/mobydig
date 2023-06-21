ARG ALPINE_VERSION=3.17
ARG ALPINE_PATCH=3
ARG GO_VERSION=1.20

FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS final
WORKDIR /src

RUN apk add --no-cache --update-cache git gcc musl-dev libcap

COPY go.mod go.sum ./
RUN go mod download

COPY ./ ./

ARG LDFLAGS="-extldflags=-static"
ARG TAGS="osusergo,netgo"
RUN go build -v -o /mobydig -tags=$"{TAGS}" -ldflags="${LDFLAGS}" ./cmd/mobydig && \
    setcap "cap_sys_admin,cap_sys_chroot,cap_sys_ptrace,cap_dac_read_search,cap_dac_override,cap_net_raw+ep" /mobydig

# Now bugger off...
USER 65534
# The ENTRYPOINT specifies the program to execute, as well as the "fixed"
# command line arguments.
ENTRYPOINT ["/mobydig"]
# CMD in combination with ENTRYPOINT now specifies *additional* default command
# line arguments, which will be added to the ENTRYPOINT parameters. These
# default arguments can be easily overriden when using "docker run <container>
# additional-args".
CMD ["neighborhood-service-a-1"]
