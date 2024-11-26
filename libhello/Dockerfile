FROM goreleaser/goreleaser-cross:v1.23.3@sha256:cc69304a8817dde675f0afe551c380545d5f34444946004961116ff394437a6f AS builder

RUN apt-get update
RUN apt-get install -y git libsdl2-dev

# credentials for fetching private (repos) deps.
# COPY ./gitcreds.sh .
# RUN ./gitcreds.sh $(cat /run/secrets/botpat)

WORKDIR /src/github.com/mpl/cgodemo/libhello/ccode
COPY ./ccode .

# building whisper cpp static library, that we're going to bind to.
# not using WORKDIR because it is (apparently) skipped on live rebuilds
RUN make

################################################
FROM scratch AS artifact
COPY --from=builder /src/github.com/mpl/cgodemo/libhello/ccode/libhello.a /libhello.a
