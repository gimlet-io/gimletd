# GimletD Design - Draft

This document is a design proposal for GimletD, a server-side component for GitOps workflows.

## Rational

The GitOps ecosystem lacks tooling to manage the GitOps repository and related workflows.

How do you store manifests in your GitOps repository, how many GitOps repositories do you have, how you model clusters, environments, teams and apps. Organizations have to answer these questions when they implement GitOps.

Gimlet CLI [answers these questions]([https://gimlet.io/gimlet-cli/concepts/](https://gimlet.io/gimlet-cli/concepts/)) by bringing conventions to the GitOps repository to help companies to implement GitOps.

It is important to see the scope of Gimlet CLI today. Its primary goal is to help developers in their local GitOps workflows and in their CI automation.

Gimlet CLI requires a local copy of the GitOps repository and while it helps following the conventions that Gimlet adds to GitOps, it doesn't enforces it.

The access to the bare GitOps repository allows developers - with or without intent - to handle the GitOps repository in an ad-hoc manner.

## GimletD provides centralized workflows to manage the GitOps repository

It acts as a release manager:

- detaches the release workflow from CI
    - unlocking advanced features
    - adding flexibility to refactor workflows

- adds central control to the release workflow
    - introduces policy based deploys
    - can perform advanced authorization
    - can enforce security standards

- while optimizes the GitOps repository write performance

Before detailing these features, first we have to break free from CI.

## Breaking free from CI

Today companies use CI to automate their releases.

Deploy and rollback steps are implemented in CI pipelines to handle the basic release workflows. Later on, further release focused steps are added: dynamic environments, cleanups, notifications.

Every application has to maintain their release steps allowing little room for centralized control, flexibility and more complex features.

GimletD introduces a new concept, the release artifact, that serves as the means to detach CI from the release workflows.

CI pipelines, instead of releasing, create an artifact for every releasable version of the application, GimletD then serves as a release manager and performs ad-hoc or policy based releases. Gimlet operates only on the releasable artifacts that CI creates.

This split allows for the above listed features.

Let's look at now the release artifact.

## The release artifact

Instead of releasing, CI generates a release artifact for each releasable version of the application.

The artifact contains all metadata that is later can be used for releasing and auditing.

```bash
{
  "id": "dev-0017d995e3-67e9d69164",
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
    "provider": "Github"
  },
  "artifacts": [
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

Gimlet CLI is getting a new command to generate the release artifact from the CI context and to push it to GimletD

```bash
gimlet artifact create --allciparams...
```

## GimletD stores the release artifacts in the artifact storage

Default artifact storage implementation will be SQL/Minio(S3)/Git

Todo performance check of git commit limiting traversing etc

go-git

git

The artifact storage stores artifacts by their id, repo name

Only support querying by id? if metadata comes from git

## Releasing a release artifact

Gimlet CLI getting further commands to perform ad-hoc release operations:

gimlet status --env --app

gimlet release --artifactid --env --app

gimlet release history --env --app --filter

gimlet rollbackto --sha --env --app

## Automated release policies

"Every artifact on master branch goes to production"