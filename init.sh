#!/usr/bin/env bash

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

CYAN='\033[0;36m'
RED='\033[0;31m'
RESET='\033[0m'

OWNER="$1"
REPO="$2"
GOVERSION="1.17"


if [[ "$1" == "" || "$2" == "" ]];
then
  printf "${RED}Error: please specify a repository name${RESET}"
  echo
  echo "Usage: init.sh <owner> <repo name>"
  echo
  exit 1
fi 

title() { printf "> $CYAN$1$RESET\r\n"; }
control() { printf "$CYAN$1$RESET\r\n"; }

deploy_git_hooks() {
  title "Deploying git hooks..."
  cp -R .githooks/. .git/hooks/
}

replace_values() {
  title "Processing $1..."
  sed -i "" "s/sha1n/$OWNER/g" "$1"
  sed -i "" "s/go-template/$REPO/g" "$1"
  sed -i "" "s/1.17/$GOVERSION/g" "$1"
}

replace_values_recursively() {
  for file in $(find "$1" -type f -iname "$2")
  do
    replace_values "$file"
  done
}

####################################################

function apply_values() {
  declare -a manual_files=("$SCRIPT_DIR/go.mod" "$SCRIPT_DIR/Makefile")
  for file in "${manual_files[@]}"
  do
    echo "$file"
    replace_values "$file"
  done 

  declare -a patterns=("*.yml" "*.md")
  for pattern in "${patterns[@]}"
  do
    replace_values_recursively "$SCRIPT_DIR" "$pattern"
  done 
}

apply_values
deploy_git_hooks

title "Running build..."
make

control "Done!"
