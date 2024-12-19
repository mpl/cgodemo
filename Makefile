.PHONY: update
update: ## updating with gok
update:
	@docker build --progress=plain -f Dockerfile.whisper --build-arg gitcreds=${GITCREDS} .

default: update

.PHONY: testupdate
testupdate: ## updating with gok
testupdate:
	@GOOS=linux GOARCH=arm64 CGO_ENABLED=1 ${GOPATH}/bin/gok update --parent_dir=./gokrazy
