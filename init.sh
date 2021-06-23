#!/usr/bin/env bash

CYAN='\033[0;36m'
RED='\033[0;31m'
RESET='\033[0m'

if [[ "$1" == "" || "$2" == "" ]];
then
  printf "${RED}Error: please specify a repository name${RESET}"
  echo
  echo "Usage: init.sh <owner> <repo name>"
  echo
  exit 1
fi 

title() {
  printf "> $CYAN$1$RESET\r\n"
}
control() {
  printf "$CYAN$1$RESET\r\n"
}
process_template() {
  title "Processing $1..."
  sed -i "" "s/sha1n/$OWNER/g" $1
  sed -i "" "s/go-template/$REPO/g" $1
}
deploy_git_hooks() {
  title "Deploying git hooks..."
  cp -R .githooks/. .git/hooks/
}


OWNER="$1"
REPO="$2"

process_template ".goreleaser.yml"
process_template "Makefile"
process_template "go.mod"
process_template ".github/dependabot.yml"
process_template ".github/workflows/readme-sync.yml"
process_template "README.md"

deploy_git_hooks

title "Running build..."
make

control "Done!"
