#!/usr/bin/env bash

CYAN='\033[0;36m'
RED='\033[0;31m'
RESET='\033[0m'

if [[ "$1" == "" ]];
then
  printf "${RED}Error: please specify a repository name${RESET}"
  echo
  echo "Usage: init.sh <repo name>"
  echo
  exit 1
fi 

title() {
  printf "> $CYAN$1$RESET\r\n"
}
control() {
  printf "$CYAN$1$RESET\r\n"
}

REPO="$1"

title "Processing Makefile..."
sed -i ".original" "s/!repo!/$REPO/" Makefile


title "Processing README.md..."
cat <<EOT > README.md
[![Build Status](https://travis-ci.com/sha1n/$REPO.svg?branch=master)](https://travis-ci.com/sha1n/$REPO) 
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/sha1n/$REPO)
[![Go Report Card](https://goreportcard.com/badge/sha1n/$REPO)](https://goreportcard.com/report/sha1n/$REPO) 
[![Release](https://img.shields.io/github/release/sha1n/$REPO.svg?style=flat-square)](https://github.com/sha1n/$REPO/releases)
![GitHub all releases](https://img.shields.io/github/downloads/sha1n/$REPO/total)
[![Release Drafter](https://github.com/sha1n/$REPO/actions/workflows/release-drafter.yml/badge.svg)](https://github.com/sha1n/$REPO/actions/workflows/release-drafter.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

# $REPO
EOT

title "Running build..."
make

control "Done!"
