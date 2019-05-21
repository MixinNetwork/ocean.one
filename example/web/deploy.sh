#!/bin/sh

rm -r dist/*
npm run dist || exit

gsutil defacl ch -u AllUsers:R gs://ocean-one

gsutil -h "Cache-Control:public, max-age=31536000" -h "Content-Encoding:gzip" -m cp dist/*.css gs://ocean-one/assets
gsutil -h "Cache-Control:public, max-age=31536000" -h "Content-Encoding:gzip" -m cp dist/*.js gs://ocean-one/assets
rm dist/*.css dist/*.js

gsutil -h "Cache-Control:public, max-age=31536000" -m cp -r dist/* gs://ocean-one/assets

scp dist/index.html one@ocean-deploy-example:tmp/ocean.one.html
ssh one@ocean-deploy-example sudo mv /home/one/tmp/ocean.one.html /var/www/html/dist/ocean.one.html
