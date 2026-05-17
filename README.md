# wanwanlog
koko hore wan wan

## Documentation

- [Design](docs/design.md)

## Release

This repository uses release-please and GoReleaser.

Use Conventional Commits on `main`:

```text
feat: add a new user-facing feature
fix: fix a user-facing bug
docs: update documentation
chore: update tooling
```

Release flow:

1. Merge changes into `main`.
2. Release Please opens or updates a release PR.
3. Merge the release PR when ready.
4. GitHub Actions creates the tag, GitHub Release, changelog, and release assets.

The current CLI version is stored in `internal/version/version.go`.

For an alpha release, create and push a prerelease tag:

```bash
git tag v0.1.0-alpha.1
git push origin v0.1.0-alpha.1
```

The tag push runs GoReleaser and marks the GitHub Release as a prerelease.

If CI should run automatically on release-please PRs, create a repository secret
named `RELEASE_PLEASE_TOKEN` with a GitHub token that has repository contents and
pull request access. Without it, the built-in `GITHUB_TOKEN` is used.
