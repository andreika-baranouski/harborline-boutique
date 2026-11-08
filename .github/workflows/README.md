# GitHub Actions Workflows

The CI for the Harborline storefront runs on GitHub-hosted runners (Ubuntu). There is no test cluster: the
storefront is a demo tenant, and nothing is deployed from this repository.

## Workflows

### Code Tests - [ci-pr.yaml](ci-pr.yaml), [ci-main.yaml](ci-main.yaml)

Run on every commit of every open pull request and on every commit to `main` or a release branch: the Go unit
tests of every Go service and the C# unit tests of `cartservice`.

### Manifest checks - [kustomize-build-ci.yaml](kustomize-build-ci.yaml), [kubevious-manifests-ci.yaml](kubevious-manifests-ci.yaml), [helm-chart-ci.yaml](helm-chart-ci.yaml), [terraform-validate-ci.yaml](terraform-validate-ci.yaml)

Build and lint the Kubernetes manifests, the Kustomize variants, the Helm chart and the Terraform under
`terraform/` on every pull request and every push to `main`. They validate files only.

The upstream deploy workflows (a staging namespace per pull request on GKE, its hourly cleanup, the release
builder) were removed on 2026-11-08: they need Google Cloud credentials this repository does not have and had
been failing on every run.
