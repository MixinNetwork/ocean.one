#!/bin/sh

while true; do
  sleep 3

  wget -q http://deploy-engine/dist/ocean.one -O /tmp/ocean.bin || continue
  mv /tmp/ocean.bin /home/one/bin/ocean.one.new
  chmod +x /home/one/bin/ocean.one.new

  if cmp -s "/home/one/bin/ocean.one" "/home/one/bin/ocean.one.new" ; then
    echo "deploy no new version"
  else
    echo "deploy new version available"
    mv /home/one/bin/ocean.one /home/one/bin/ocean.one.old
    mv /home/one/bin/ocean.one.new /home/one/bin/ocean.one
    sudo systemctl restart ocean-one-http.service
  fi
done
