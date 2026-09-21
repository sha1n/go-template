[![Go](https://github.com/sha1n/go-template/actions/workflows/go.yml/badge.svg)](https://github.com/sha1n/go-template/actions/workflows/go.yml)
[![codecov](https://codecov.io/gh/sha1n/go-template/graph/badge.svg)](https://codecov.io/gh/sha1n/go-template)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/sha1n/go-template)
[![Release](https://img.shields.io/github/release/sha1n/go-template.svg?style=flat-square)](https://github.com/sha1n/go-template/releases)
![GitHub all releases](https://img.shields.io/github/downloads/sha1n/go-template/total)
[![Release Drafter](https://github.com/sha1n/go-template/actions/workflows/release-drafter.yml/badge.svg)](https://github.com/sha1n/go-template/actions/workflows/release-drafter.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

# go-template

## Before anything else

Create your repository with GitHub's **["Use this template"](https://github.com/sha1n/go-template/generate)** button — **do not `git clone` this template directly.** "Use this template" starts your new repo with a clean, single-commit history; cloning would drag along this template's entire commit history.

Then clone *your new repository* and initialize it:
```bash
git clone git@github.com:<owner>/<repo>.git
cd <repo>

# With Claude Code (recommended): guided, tailored, self-cleaning
/init-template

# Or plain, deterministic rename (Go toolchain only):
make init OWNER=<owner> REPO=<repo> GOVERSION=<x.y>
```

## Features

- Guided/deterministic project initialization (`/init-template` or `make init`)
- Makefile
  - standard build/test/format/lint
  - protobuf support with repo private `protoc` installation (see `PROTOC_VERSION` in [Makefile](Makefile))
  - multi-platform binaries
  - goreleaser with Homebrew cask support (publishing the cask needs a `HOMEBREW_TAP_GITHUB_TOKEN` repository secret with write access to the tap)
- Workflows
  - Go build + coverage - [go.yml](/.github/workflows/go.yml)
  - Release Drafter - [release-drafter.yml](/.github/workflows/release-drafter.yml)
  - Dependabot App - [dependabot.yml](/.github/dependabot.yml)
- Jekyll site setup with the [Cayman](https://github.com/pages-themes/cayman) theme (and some color overrides)
