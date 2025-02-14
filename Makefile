# top level makefile that starts the builds with docker.
# once within docker, ./whisper/Makefile is where is action is at.

.PHONY: update
update: ## push update to GUS server
update:
	@docker build --progress=plain -f Dockerfile.whisper --build-arg gitcreds=${GITCREDS} --target gaf --output=. .
	@./bin/ingest.sh

.PHONY: forceupdate
forceupdate: ## push update directly to device, usually not what you want.
forceupdate:
	@docker build --progress=plain -f Dockerfile.whisper --build-arg gitcreds=${GITCREDS} --target forceupdate .

.PHONY: updatenodocker
updatenodocker: ## push update to GUS server.
updatenodocker:
	@make gaf
	@./bin/ingest.sh

.PHONY: gaf
gaf: ## create gaf for ingestion
gaf:
	@CGO_ENABLED=1 GOOS=linux GOARCH=arm64 ${GOPATH}/bin/gok --parent_dir ./gokrazy/ overwrite --gaf /tmp/full.gaf

.PHONY: forceupdatenodocker
forceupdatenodocker: ## push update directly to device, usually not what you want.
forceupdatenodocker:
	# TODO: do it in two steps too?
	@CGO_ENABLED=1 GOOS=linux GOARCH=arm64 CGO_ENABLED=1 ${GOPATH}/bin/gok update --parent_dir=./gokrazy

default: update
