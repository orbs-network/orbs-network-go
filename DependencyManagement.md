# Dependency Management

## Basic information

This project uses git submodules for dependency management, which means that we do not store third-party source code in our source tree, but instead point to a different git repositories from `vendor/website/vendor-name/lib-name` folder.

Git stores a link to a specific commit in a different repository. That way we always have a specific version of a dependency what we use to raise both the stability and security of our codebase.

Most of the time you only want to fetch the dependencies via `./git-submodule-checkout.sh` script. It will both init and update you to the latest version (set to the specific version that was vetted and selected) of all dependencies.

In case you need to introduce or update a dependency, use `manul` to update a commit that points to the dependency version, and open a pull request. (https://github.com/kovetskiy/manul)

After the pull request is merged, everyone has to fetch the dependencies using `./git-submodule-checkout.sh`. You can always verify if you are up to date with command `manul -Q`. If you see a plus sign (`+`) next to a commit id, it means that it's out of sync.

## Initial flow

To install all dependencies after checking out the project, run:

`./git-submodule-checkout.sh`

To install dependency management tool, run:

`go get github.com/kovetskiy/manul`

The dependency management tool will enable you to update to specific versions and add new dependencies.

## Adding new dependency

Add new dependency from `master`, or 'latest version':

`manul -I github.com/username/repo`

Add new dependency pointing to a certain commit:

`manul -I github.com/username/repo=COMMIT-HASH`


Commit and push the resulting changes to `.gitmodules` file and the `/vendor/path/to/your/dependency` link.

After you commit and push, verify the new dependecy appears under /vendor on github.com (make sure to switch to your branch to see it).



## Updating dependency to point to a certain commit

Running `manul -U github.com/username/repo=COMMIT-HASH` will change the submodules version to the one you just set. Note that this will automatically checkout the correct version as well, so after running the update command you are in actual working with the updated version.

## Updating dependency to point to the latest 'master' commit

Running `manul -U github.com/username/repo` - without the commit hash, will change the submodules version to the latest commit in the master of that repo. Note that this will automatically checkout the correct version as well, so after running the update command you are in actual working with the updated version.

## Updating dependencies after pulling from the upstream

The checkout script resets to the committed state, this works both when doing the first init or at any later stage. So to update from pull run:

`./git-submodule-checkout.sh`

## Rolling back

In cases where you made changes just to check a specific version (commit hash) of a dependency (using `manul -U`), rolling back or resetting to the committed state is done also via the checkout script:

`./git-submodule-checkout.sh`