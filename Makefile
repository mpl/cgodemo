# top level makefile that starts the builds with docker.
# once within docker, ./whisper/Makefile is where is action is at.

.PHONY: update
update: ## push update to GUS server
update:
	@docker build --progress=plain -f Dockerfile.whisper --build-arg gitcreds=${GITCREDS} --target update .

.PHONY: forceupdate
forceupdate: ## push update directly to device, usually not what you want.
forceupdate:
	@docker build --progress=plain -f Dockerfile.whisper --build-arg gitcreds=${GITCREDS} --target forceupdate .

.PHONY: updatenodocker
updatenodocker: ## push update to GUS server.
updatenodocker:
	@./bin/ingest.sh

.PHONY: forceupdatenodocker
forceupdatenodocker: ## push update directly to device, usually not what you want.
forceupdatenodocker:
	@CGO_ENABLED=1 GOOS=linux GOARCH=arm64 CGO_ENABLED=1 ${GOPATH}/bin/gok update --parent_dir=./gokrazy

default: update
