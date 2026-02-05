#!/bin/sh

cat registry_token.txt | docker login docker.git.sos.ethz.ch --password-stdin --username vmwiz
