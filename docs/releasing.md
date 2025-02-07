# Releasing CAPMVM

> IMPORTANT - before doing a release that updates the major or minor version make sure you have updated and commited [metadata.yaml](https://github.com/liquidmetal-dev/cluster-api-provider-microvm/blob/main/metadata.yaml) with the new version.

## Determine release version

The projects follows [semantic versioning](https://semver.org/#semantic-versioning-200)
and so the release version must adhere to this specification.
Depending on the changes in the release you will need to decide the next
appropriate version number.

Its advised that you pull the tags and view the latest release (i.e. tag):

```bash
git pull --tags

git describe --tags --abbrev=0
```

## Update `metadata.yaml`

If you are increasing by a major or minor version, edit `metadata.yaml` to contain
the new contract.

Open a PR and merge to main BEFORE you continue with the release.

## Create tag

* Checkout upstream main
* Create a tag with the version number:

```bash
RELEASE_VERSION=v0.1.0
```

Then:

```bash
git tag -s "${RELEASE_VERSION}" -m "${RELEASE_VERSION}"
```

* Push the tag (to upstream if working from a fork)

``` bash
git push origin "${RELEASE_VERSION}"
```

* Check the [release](https://github.com/liquidmetal-dev/cluster-api-provider-microvm/actions/workflows/release.yml) GitHub Actions workflow completes successfully.
* Check that the [docker image](https://github.com/orgs/liquidmetal-dev/packages?repo_name=cluster-api-provider-microvm) for that tag was created successfully. (This
won't actually be visible while the repo is private for... reasons.)

## Edit & Publish GitHub Release

* Go to the draft release in GitHub.
* Check that the assets were attached correctly
* Make any edits to generated release notes
  * Note which versions of Flintlock are compatible with this release
  * If there are any breaking changes then manually add a note at the beginning
    of the release notes informing the user what they need to be aware of/do.
  * Sometimes you may want to combine changes into 1 line
* If this is **not** a pre-release untick `This is a pre-release`
* Check that the box to generate a discussion is ticked, and that the discussion
  goes into 'Announcements'.
* Publish the draft release and when asked say yes to creating a discussion.

## Announce release

When the release is available announce it in the #liquid-metal slack channel.

## Update the version compatibility table

Once the release is published, edit [docs/compatibility.md](docs/compatibility.md)
and update the table to contain the new version and any compatible Flintlock versions.
Open a PR and merge the changes.
