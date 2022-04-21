#!/usr/bin/env bash

GREEN='\033[0;32m'
RED='\033[0;31m'
RESET='\033[0m'

PB_REL="https://github.com/protocolbuffers/protobuf/releases"


PROTOC_VERSION="$1"
TARGET_DIR=$2
XMARK=$'✘'
VMARK=$'✔'
eval `go env`

download_protoc() {

  title "Downloading protoc for $GOOS/$GOARCH..."

  if [[ "$GOOS" == "darwin" && "$GOARCH" == "amd64" ]]; 
  then
    osarch="osx-x86_64"
  elif [[ "$GOOS" == "linux" && "$GOARCH" == "amd64" ]]; 
  then
    osarch="linux-x86_64"
  elif [[ "$GOOS" == "windows" && "$GOARCH" == "amd64" ]]; 
  then
    osarch="win64"
  else
    error "unsupported platform '$GOOS/$GOARCH'"
    exit 1
  fi 

  tmp_dir=$(mktemp -d 2>/dev/null || mktemp -d -t 'protoc-download')
  file_name="protoc-${PROTOC_VERSION}-${osarch}.zip"
  file_path="${tmp_dir}/${file_name}"

  download_url="${PB_REL}/download/v${PROTOC_VERSION}/${file_name}"
  curl -LsSf --compressed -o "${file_path}" "${download_url}" && \
    ok "protoc package successfully downloaded to ${file_path}" && \
    title "extracting..." && \
    unzip "${file_path}" -d "${TARGET_DIR}" && \
    ok "protoc package extracted successfully" && \
    chmod +x "${TARGET_DIR}/bin/protoc" && \
    ok "protoc installed to ${TARGET_DIR}" ; \
    title "cleaning up..." && \
    rm -rf "${tmp_dir}"
}

error() {
  printf "  ${RED}${XMARK}${RESET}  $1\r\n"
}

ok() {
  printf "  ${GREEN}${VMARK}${RESET}  $1\r\n"
}

title() {
  printf "  -  $1\r\n"
}

if [ "$#" -lt 2 ]; then
  error "required arguments are missing!"

  printf "
  Usage ./setup_protoc.sh <PROTOC_VERSION> <TARGET_DIR>

"
  exit 1
fi

if [[ -d "${TARGET_DIR}" ]]
then 
  error "target directory already exists - skipping..."
  exit 0
fi 

if download_protoc ; then
  ok "protoc successfully installed to ${TARGET_DIR}"
else
  error "installation failed"
  exit 1
fi

