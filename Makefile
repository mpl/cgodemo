.PHONY: update
update: ## updating with gok
update:
	@docker build --progress=plain -f Dockerfile.whisper --build-arg gitcreds=${GITCREDS} .

default: update
