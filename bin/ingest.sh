#!/bin/sh
set -e

# CGO_ENABLED=1 GOOS=linux GOARCH=arm64 ${GOPATH}/bin/gok --parent_dir ./gokrazy/ overwrite --gaf ./full.gaf
SBOM=$(unzip -p ./full.gaf sbom.json | jq -r '.sbom_hash')
echo $SBOM
LINK=$(GOOS=linux GOARCH=arm64 ${GOPATH}/bin/gok --parent_dir ./gokrazy/ push --gaf ./full.gaf --server http://100.109.9.11:8655 --json | jq -r '.download_link')
echo $LINK
MACHINE_ID=$(cat ./gokrazy/hello/config.json | jq -r '.PackageConfig."github.com/gokrazy/gokrazy/cmd/randomd".ExtraFileContents."/etc/machine-id"' | xargs)
echo $MACHINE_ID
curl -sL -d "{\"machine_id_pattern\": \"${MACHINE_ID}\", \"sbom_hash\": \"${SBOM}\" , \"registry_type\": \"localdisk\", \"download_link\": \"${LINK}\" }" -X POST http://100.109.9.11:8655/api/v1/ingest
