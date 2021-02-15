# GitLab Releases Resource

Fetches and creates versioned GitLab releases.   Note that `check` will skip tags that do not have associated releases.

:warning: Limitations :warning:

Gitlab has a [known bug](https://gitlab.com/gitlab-org/gitlab/-/issues/28978) that makes impossible to download assets published to a project using a private-token. When using `in` with such release, download assets will contain the plain HTML of gitlab's sign-in page. 

Once the bug will be fixed, the resource will behave as expected with no further modification.

## Source Configuration

* `repository`: *Required.* The repository name that contains the releases.

* `access_token`: *Required.* Used for accessing a release in a private-repo
   during an `in` and pushing a release to a repo during an `out`. The access
   token you create is only required to have the `repo` or `public_repo` scope.

* `gitlab_api_url`: *Optional.* If you use a non-public GitLab deployment then
  you can set your API URL here.

* `insecure`: *Optional. Default `false`.* When set to `true`, concourse will allow
  insecure connection to your gitlab API.

* `tag_filter`: *Optional.* If set, override default tag filter regular
  expression of `v?([^v].*)`. If the filter includes a capture group, the capture
  group is used as the release version; otherwise, the entire matching substring
  is used as the version.

### Example

``` yaml
- name: gl-release
  type: gitlab-release
  source:
    repository: group/project
    access_token: abcdef1234567890
```

``` yaml
- get: gl-release
```

``` yaml
- put: gl-release
  params:
    tag: path/to/tag/file
    body: path/to/body/file
    globs:
    - paths/to/files/to/upload-*.tgz
```

To get a specific version of a release:

``` yaml
- get: gl-release
  version: { tag: 'v0.0.1' }
```

To set a custom tag filter:

```yaml
- name: gl-release
  type: gitlab-release
  source:
    owner: concourse
    repository: concourse
    tag_filter: "version-(.*)"
```

## Behavior

### `check`: Check for released versions.

Releases are listed and sorted by their tag, using
[semver](http://semver.org) semantics if possible. If `version` is specified, `check` returns releases from the specified version on. Otherwise, `check` returns the latest release.

### `in`: Fetch assets from a release.

Fetches artifacts from the given release version. If the version is not
specified, the latest version is chosen using [semver](http://semver.org)
semantics.

Also creates the following files:

* `tag` containing the git tag name of the release being fetched.
* `version` containing the version determined by the git tag of the release being fetched.
* `body` containing the body text of the release.
* `commit_sha` containing the commit SHA the tag is pointing to.

#### Parameters

* `globs`: *Optional.* A list of globs for files that will be downloaded from
  the release. If not specified, all assets will be fetched.
* `include_sources`: *Optional.* A list of source format to download from
  the release. If not specified, no sources will be fetched. eg. `["zip", "tar.gz",
  "tar.bz2", "tar"]`
* `include_source_tarball`: *Optional.* Enables downloading of the source artifact tarball
   for the release as `source.tar.gz`.  Defaults to `false`. Equivalent to
   `include_sources: ["tar.gz"]`.
* `include_source_zip`: *Optional.* Enables downloading of the source artifact tarball
   for the release as `source.zip`.  Defaults to `false`. Equivalent to
   `include_sources: ["zip"]`.

### `out`: Publish a release.

Given a `commit_sha` and  `tag`, this tags the commit and creates a release on GitLab, then uploads the files
matching the patterns in `globs` to the release.

#### Parameters

* `commitish`: *Optional, if tag is not specified.* A path to a file containing the commitish (SHA, tag,
  branch name) that the new tag and release should be associated with.

* `tag`: *Required.* A path to a file containing the name of the Git tag to use
  for the release.

* `tag_prefix`: *Optional.*  If specified, the tag read from the file will be
prepended with this string. This is useful for adding v in front of version numbers.

* `name`: *Optional.* A path to a file containing the name of the release. Defaults to `tag` value.
  for the release.

* `body`: *Optional.* A path to a file containing the body text of the release.

* `globs`: *Optional.* A list of globs for files that will be uploaded alongside
  the created release.

## Development

### Prerequisites

* golang is *required* - version 1.15.x is tested; earlier versions may also
  work.
* docker is *required* - version 19.03.x is tested; earlier versions may also
  work.
* go mod is used for dependency management of the golang packages.

### Running the tests

The tests have been embedded with the `Dockerfile`; ensuring that the testing
environment is consistent across any `docker` enabled platform. When the docker
image builds, the test are run inside the docker container, on failure they
will stop the build.

Run the tests with the following command:

```sh
docker build -t gitlab-release-resource .
```

### Contributing

Please make all pull requests to the `master` branch and ensure tests pass
locally.

### Credits

This project was initialy created by @edtan and forked from [edtan/gitlab-release-resource](https://github.com/concourse/github-release-resource) which is no longer maintained. It has been re-imported to get rid of the fork relationship to the repository [concourse/github-release-resource](https://github.com/concourse/github-release-resource).


