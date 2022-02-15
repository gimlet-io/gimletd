**ARCHIVED**

**Merged into https://github.com/gimlet-io/gimlet-cli**

**The docker image location is still `ghcr.io/gimlet-io/gimletd:latest`**

**Look for future releases under https://github.com/gimlet-io/gimlet-cli/releases tagged with `gimletd-vx.y.z`**


# gimletd - the GitOps release manager

[![Gitpod ready-to-code](https://img.shields.io/badge/Gitpod-ready--to--code-blue?logo=gitpod)](https://gitpod.io/#https://github.com/gimlet-io/gimletd)
[![Go Report Card](https://goreportcard.com/badge/github.com/gimlet-io/gimlet-cli)](https://goreportcard.com/report/github.com/gimlet-io/gimletd)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

GimletD acts as a release manager and detaches the release workflow from CI. By doing so, it unlocks the possibility of advanced release logics and flexibility to refactor workflows.

By assuming all release related work, it adds central control to the release workflow by introducing policy based deploys and advanced authorization and security standards, while it also optimizes the GitOps repository write performance.

See the [documentation](https://gimlet.io/gimletd/getting-started/).

## Contribution Guidelines

Thank you for your interest in contributing to the Gimlet project.

Below are some of the guidelines and best practices for contributing to this repository:

### New Features / Components

If you have any ideas on new features or want to improve the existing features, you can suggest it by opening a [GitHub issue](https://github.com/gimlet-io/gimletd/issues/new). Make sure to include detailed information about the feature requests, use cases, and any other information that could be helpful.]

### Developing GimletD

GimletD provides a preconfigured Gitpod development environment.

If you have not tried Gitpod yet, you really should. Click this button [![Gitpod ready-to-code](https://img.shields.io/badge/Gitpod-ready--to--code-blue?logo=gitpod)](https://gitpod.io/#https://github.com/gimlet-io/gimletd) to get a cloud based development environment instantly.

#### Initial setup

`.gitpod.yml` has the automation to be able to run GimletD in a debug session.

GimletD integrates with Github through deploy keys.

Set the following Gitpod variables so Gitpod can create a `.env` file and a `deploykey` file for you on startup:

- GITOPS_REPO
- DEPLOY_KEY
- DEPLOY_KEY_PUB

Use the `ssh-keygen -a 100 -t ed25519 -C your@email.here -f $(pwd)/deploykey` command to generate a deploykey when your setup GimletD for the first time.
Use the `sed -z 's/\n/\\n/g' deploykey | base64 -w 0` command to get a base64 encoded representation of the SSH key that you store as the DEPLOY_KEY Gitpod variable
