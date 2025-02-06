#!/bin/sh
set -e

GOOS=linux GOARCH=arm64 ${GOPATH}/bin/gok --parent_dir ./gokrazy/ overwrite --gaf /tmp/full.gaf 
SBOM=$(unzip -p /tmp/full.gaf sbom.json | jq -r '.sbom_hash')
echo $SBOM
LINK=$(GOOS=linux GOARCH=arm64 ${GOPATH}/bin/gok --parent_dir ./gokrazy/ push --gaf /tmp/full.gaf --server http://192.168.0.29:8655 --json | jq -r '.download_link')
echo $LINK
MACHINE_ID=$(cat ./gokrazy/hello/config.json | jq -r '.PackageConfig."github.com/gokrazy/gokrazy/cmd/randomd".ExtraFileContents."/etc/machine-id"' | xargs)
echo $MACHINE_ID
curl -sL -d "{\"machine_id_pattern\": \"${MACHINE_ID}\", \"sbom_hash\": \"${SBOM}\" , \"registry_type\": \"localdisk\", \"download_link\": \"${LINK}\" }" -X POST http://192.168.0.29:8655/api/v1/ingest
