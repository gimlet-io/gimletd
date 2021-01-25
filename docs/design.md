# GimletD - the GitOps release manager

This document is a design proposal for GimletD, a server-side release manager component for GitOps workflows.

## Rational

The GitOps ecosystem lacks tooling to manage the GitOps repository and related workflows.

- How do you store manifests in your GitOps repository?
- how many GitOps repositories do you have
- how do you model clusters, environments, teams and apps?

Organizations have to answer these questions when they implement GitOps.

Gimlet CLI [answers these questions]([https://gimlet.io/gimlet-cli/concepts/](https://gimlet.io/gimlet-cli/concepts/)) by bringing conventions to the GitOps repository to help companies implementing GitOps.
But it is important to see the scope of Gimlet CLI today: its goal is to help developers in their local GitOps workflows and in their CI automation.

![GimletCLI used from CI](https://gimlet.io/gitops-with-ci.png)

Gimlet CLI requires a local copy of the GitOps repository and while it helps following the conventions that Gimlet adds to GitOps, it doesn't enforces it.
The access to the bare GitOps repository allows developers - with or without intent - to handle the GitOps repository in an ad-hoc manner.

## GimletD provides centralized workflows to manage the GitOps repository

GimletD acts as a release manager and detaches the release workflow from CI. By doing so, it unlocks the possibility of advanced release logics and flexibility to refactor workflows.

By assuming all release related work, it adds central control to the release workflow by introducing policy based deploys and advanced authorization and security standards, while it also optimizes the GitOps repository write performance.

## Breaking free from CI

Today companies use CI to automate their releases.

Deploy and rollback steps are implemented in CI pipelines to handle the basic release workflows.
Later on, further release focused steps are added: dynamic environments, cleanups, notifications
that every application has to maintain. This decentralized approach allows little room for control, flexibility and complex features.

GimletD assumes all release focused tasks and the management of the GitOps repository.

GimletD achieves this by introducing a new concept, the release artifact, that serves as the means to detach the release workflows from CI.

With GimletD, instead of releasing, CI pipelines create an artifact for every releasable version of the application,
GimletD then serves as a release manager to perform ad-hoc or policy based releases.

Gimlet operates only on the releasable artifacts that CI creates. This split allows for the above listed features.

![GimletD operates on the release artifacts, manages the GitOps repository](https://gimlet.io/gimletd-with-gitops.png)

Now, let's look at the release artifact.

## The release artifact

Instead of releasing, CI generates a release artifact for each releasable version of the application. The artifact contains all metadata that can be later used for releasing and auditing.

The release artifact idea is adopted from [Lunar's release manager](https://github.com/lunarway/release-manager) project.

```json
{
  "id": "example-service-017d995e32e3d1998395d971b969bcf682d2085",
  "version": {
    "sha": "017d995e32e3d1998395d971b969bcf682d2085",
    "branch": "master",
    "pr": true,
    "source_branch": "feature/x", 
    "authorName": "First Last",
    "authorEmail": "email@email.com",
    "committerName": "First Last",
    "committerEmail": "email@email.com",
    "message": "reformat something",
    "name": "example-service",
    "url": "https://github.com/owner/repo/commits/0017d995e32e3d1998395d971b969bcf682d2085",
  },
  "context": {
    [...arbitrary environment variables from CI...]  
  },
  "environments": [
    {
      "name": "staging",
      [...the complete set of Gimlet environments from the Gimlet environment files...]  
    },    
  ],
  "items": [
    {
      "name": "ci",
      "jobUrl": "https://jenkins.example.com/job/dev/84/display/redirect",
    },
    {
      "name": "image",
      "repository": "quay.io/example.com/example",
      "tag": "017d995e32e3d1998395d971b969bcf682d2085"
    },
  ]
}
```

Gimlet CLI is getting three new commands to create the artifact.

- `gimlet artifact create` is initiating the artifact json.
- `gimlet artifact add` adds a new artifact item to the artifact json file. As the CI pipeline progresses, with this command you can append new artifacts to it: CI job information, test results, Docker image information, etc
- `gimlet artifact push` is pushing the final artifact to the GimletD server. Typically, the final step of the CI pipeline.

GimletD then stores the artifacts in the artifact store.

## GimletD stores the release artifacts in the artifact storage

The artifact store will be backed by an RDBMS, and GimletD provides and API to browse the artifacts.

## Releasing a release artifact

GimletD accepts ad-hoc commands to release a release artifact:

`gimlet release --artifact-id --env`

and accepts automated release policies to perform releases based on rules:

`gimlet policy auto-release --app --event --branch --env`

(This proposal is also published at https://gimlet.io/blog/gimletd-the-gitops-release-manager/)
