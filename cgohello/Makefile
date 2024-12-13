GO111MODULE 	?= on
export GO111MODULE

GOPATH			?= $(shell go env GOPATH)

# We want to make sure that we use the same source for whisper when building
# libwhisper.a and when depending on in our Go code.
# So we have a replace directive in our go.mod, to ./whisper.cpp
# We vendor it for now, because it's the simplest solution that works both
# for local dev, and for in prod. But might do smarter things instead later.
LIBHELLO = ${PWD}/../libhello
WHISPER_CPP = ${LIBHELLO}/ccode
LIBWHISPER = ${WHISPER_CPP}/libhello.a

# otoh, since we're building from the same place, when we're switching arch,
# since we're going to remove the intermediate build objects, we don't want to
# overwite the binaries, so give them different names.
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
    TRANSCRIBE ?= devhello
else
    TRANSCRIBE ?= hello
endif

.PHONY: all
all: build

.PHONY: clean
clean: ## Clean.
clean:
	@rm ./bin/${TRANSCRIBE} && cd ${WHISPER_CPP} && make clean

.PHONY: cleanall
cleanall: ## Cleanall.
cleanall: clean
	@rm ./bin/${TRANSCRIBE}

.PHONY: vendor
vendor: ## Vendor.
vendor:
	@./bin/vendor.sh

.PHONY: build
build: ## Build.
build: ${LIBWHISPER} ./bin/${TRANSCRIBE}
	@echo ">> Building"

.PHONY: transcribe
transcribe: ## transcribe audio from network
transcribe: ./bin/${TRANSCRIBE}
	@echo ">> Running transcribe"
	@./bin/${TRANSCRIBE} -source=http://localhost:8362 -host=:8361 -modelpath=./data/models/ggml-base.en.bin -stdout=true -v=true -gcgrace=120 -gcinterval=60

${LIBWHISPER}:
#	@cd ${WHISPER_CPP} && make
	# N.B: ${PWD} does not change with cd
	@cd ${LIBHELLO} && docker build --progress=plain --target=artifact --output type=local,dest=${WHISPER_CPP}/ .

.PHONY: ./bin/${TRANSCRIBE}
./bin/${TRANSCRIBE}: # build the binary
./bin/${TRANSCRIBE}: ${LIBWHISPER}
	@mkdir -p bin
#	@C_INCLUDE_PATH=${WHISPER_CPP} LIBRARY_PATH=${WHISPER_CPP} CGO_ENABLED=1 go build -o ./bin/${TRANSCRIBE} ./cmd/main.go
	GOOS=linux GOARCH=arm64 CC=aarch64-linux-musl-gcc CGO_ENABLED=1 go build -ldflags="-extldflags '-static -L ${WHISPER_CPP}'" -o ./bin/${TRANSCRIBE} ./cmd/main.go

.PHONY: gok
gok: ## Deploy with gokrazy
gok: ${LIBWHISPER}
	# TODO: var or relpath for -L in gok config
	GOOS=linux GOARCH=arm64 CC=aarch64-linux-musl-gcc CGO_ENABLED=1 gok update

.PHONY: glass
glass: ## Breakglass tarball.
glass: ./bin/${TRANSCRIBE}
	@cd ./bin && tar cf ../breakglass.tar ${TRANSCRIBE}

.PHONY: breakglass
breakglass: ## breaking glass
breakglass: glass
	@breakglass -debug_tarball_pattern=breakglass.tar hello

.PHONY: prelint
prelint: ## Prepare for linting.
# Note: looks like just ${LIBWHISPER} (instead of build), is not enough, as it does not make whisper.h available or something.
prelint: build
	@echo ">> Prepare deps for linting"

default: ./bin/${TRANSCRIBE}
