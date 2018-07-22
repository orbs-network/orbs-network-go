# Dependency Management

## Basic information

This project uses git submodules for dependency management, which means that we do not store third-party source code in our source tree, but instead point to a different git repositories from `vendor/website/vendor-name/lib-name` folder.

Git stores a link to a certain commit in a different repository. That way we always have a certain version and can make sure that we always have working codebase.

Most of the time you only want to fetch the dependencies via `./git-submodules-checkout.sh` script.

In case you need to introduce or update a dependency, use `manul` to update a commit that points to the dependency version, and open a pull request.

After the pull request is merged, everyone has to fetch the dependencies using `./git-submodules-checkout.sh`. You can always verify if you are up to date with command `manul -Q`. If you see a plus sign (`+`) next to a commit id, it means that it's out of sync.

## Initial flow

To install all dependencies after checking out the project, do

`./git-submodules-checkout.sh`

To install dependency management tool, do

`go get github.com/kovetskiy/manul`

## Adding new dependency

Add new dependency from `master`:

`manul -I github.com/username/repo`

Add new dependency pointing to a certain commit:

`manul -I github.com/username/repo=COMMIT`

## Updating dependency to point to a certain commit

`manul -U github.com/username/repo=COMMIT`

`./git-submodules-checkout.sh`

## Updating dependencies after pulling from origin

`./git-submodules-checkout.sh`

## Rolling back

`./git-submodules-checkout.sh`