# gimletd - the GitOps release manager

GimletD acts as a release manager and detaches the release workflow from CI. By doing so, it unlocks the possibility of advanced release logics and flexibility to refactor workflows.

By assuming all release related work, it adds central control to the release workflow by introducing policy based deploys and advanced authorization and security standards, while it also optimizes the GitOps repository write performance.

Read the design docs [here](docs/design.md).

## Docker image

```bash
docker run -it ghcr.io/gimlet-io/gimletd:latest
```

## Running in Kubernetes

```bash
cat << EOF > values.yaml
image:
    repository: ghcr.io/gimlet-io/gimletd
    tag: latest
    
volumes:
  - name: data
    path: /var/lib/gimletd
    size: 10Gi
    storageClass: default
EOF

helm template gimletd onechart/onechart -f values.yaml
```

with ingress and TLS:

```bash
cat << EOF > values.yaml
containerPort: 8888
image:
  repository: ghcr.io/gimlet-io/gimletd
  tag: latest
volumes:
  - name: data
    path: /var/lib/gimletd
    size: 10Gi
    storageClass: default
ingress:
  annotations:
    kubernetes.io/ingress.class: nginx
    cert-manager.io/cluster-issuer: letsencrypt
  host: gimletd.my-company.com
  tlsEnabled: true
EOF

helm template gimletd onechart/onechart -f values.yaml
```

## First start

When you first start GimletD, it inits a file based SQLite3 database, and prints the admin token to the logs.

Use this token to create a regular user token:

```bash
curl -i \
    -H "Content-Type: application/json" \
    -H "Accept: application/json" \
    -X POST -d '{"login":"laszlo"}' \
    http://localhost:8888/api/user?access_token=$GIMLET_TOKEN
```

Save the returned user token from the result.
