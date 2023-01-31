# Release Management

## Release Process

the flow should be as follows

1. Each release is developed on a branch that is derived from the development branch (e.g. development-2.0).
2. All pull requests should be made to this release branch; no commits should be made directly to the development branch.
3. The development branch should always reflect the latest release.
4. the release branch should go to development then to master.
5. The latest commit should be tagged, tags should be `annotated` and in the format `v*` for the release workflow to work. example `git tag -a v*`
6. Our github workflow `.github/workflows/release.yml` is triggered when a tag is pushed.
7. The workflow basicly runs goreleaser to create a new release which creates the different release assets required to run the provider on different platforms `i.e windows,linux and mac`.
8. the release.yml requires some secrets to be existed on the repo.
    - `GPG private key`: this is generated with the command `gpg --armor --export-secret-keys [key ID or email]`.
    - `PASSPHRASE`: The passphrase for your GPG private key.
9. Once the release is done the terraform registry always watches the repo for new releases and when it finds a new release it publishes a new provider version.

## Releasing for each environment

For example, releasing v1.7.0 for different networks, the tags should be:

1. `dev`: v1.7.0-dev
2. `qa`: v1.7.0-qa
3. `test`: v1.7.0-rcX
4. `main`: v1.7.0

### Updating dependencies

The Grid Terraform provider depends on ZOS and the Substrate client, so new releases of these repos should be reflected in this project.

#### Updating zos version

This basicly done as any go dependency update the required version in go.mod and if there are changes required on the code should be fixed for example if we want to update to the last commit in zos which is `d9c7fe2` we do the following.

```bash
 go get https://github.com/threefoldtech/zos@d9c7fe2
```

#### Updating substrate client

1. In the current design we have substrate pkg for each environment as submodules(this should be changed soon, we should use provider versions instead).
2. When we want to update substrate-client for any environment for example for `dev` environment we go to `pkgs/substrates/substrate-dev` and checkout the commit we want to update to and then we do the required changes on code.
3. substrate client depends on `go-substrate-rpc-client` so consider updating it if needed
    for example if we are going to update to the commit `c3a7ge8`

```bash
go get github.com/threefoldtech/go-substrate-rpc-client/v4@c3a7ge8
```

### Known issues

The approach of having susbstrate-client for each environment is not good because

- We need to manage different substrate versions and repeat the same changes every time we need to update an evironment.
- Also sometime the interface itself go changed and hence we can not use this approach.
- This sometimes required to update `go-substrate-rpc-client` version is substrate client is rebased or so for example.
