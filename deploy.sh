#!/bin/sh

grep -q '"production"' config/config.go || exit
grep -q '"aaff5bef-42fb-4c9f-90e0-29f69176b7d4"' config/config.go || exit
sed -i --  "s/BUILD_VERSION/`git rev-parse HEAD`/g" config/config.go || exit
go build -ldflags="-s -w" || exit
rm config/config.go

scp ocean.one one@ocean-deploy-engine:tmp/ocean.one || exit
ssh one@ocean-deploy-engine sudo mv /home/one/tmp/ocean.one /var/www/html/dist/ocean.one || exit
