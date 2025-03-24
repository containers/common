# Contributing to Containers Projects: Go Language Guidelines

This is an appendix to the main [Contributing Guide](./CONTRIBUTING.md) and is intended to be read after that document.
It contains guidelines and general rules for contributing to projects under the Containers org that are written in the Go language.
At present, this means the following repositories:

- [podman](https://github.com/containers/podman)
- [buildah](https://github.com/containers/buildah)
- [skopeo](https://github.com/containers/skopeo)
- [common](https://github.com/containers/common)
- [image](https://github.com/containers/image)
- [storage](https://github.com/containers/storage)
- [libhvee](https://github.com/containers/libhvee)
- [psgo](https://github.com/containers/psgo)

## Topics

* [Go Dependency updates](#go-dependency-updates)
* [Testing changes in a dependent repository](#testing-changes-in-a-dependent-repository)
* [git bisect a change in a Go dependency](#git-bisect-a-change-in-a-go-dependency)

## Unit Tests

Unit tests for Go code are added in a separate file within the same directory, named `..._test.go` (where the first part of the name is often the name of the file whose code is being tested).
Our Go projects to not require unit tests, but contributors are strongly encouraged to unit test any code that can have a reasonable unit test written.

## Go Dependency updates

To automatically keep dependencies up to date we use the [renovate](https://github.com/renovatebot/renovate) bot.
The bot automatically opens new PRs with updates that should be merged by maintainers.

However sometimes, especially during development, it can be the case that you like to update a dependency.

To do so you can use the `go get` command, for example to update containers/storage to the a specific version use:
```
$ go get github.com/containers/storage@v1.55.1
```

Or to update it to the latest commit from main use:
```
$ go get github.com/containers/storage@main
```

This command will update the go.mod/go.sum files, because we use [go's vendor mechanism](https://go.dev/ref/mod#vendoring)
you must also update the files in the vendor dir. To do so use
```
$ make vendor
```

Then commit the changes and open a PR. If you want to add other changes it is recommended to keep the
dependency updates in their own commit as this makes reviewing them much easier.

Note when cutting a new release always make sure we only use tagged version of our own containers/...
dependencies to ensure all our tools use the same properly tested library versions.

## Testing changes in a dependent repository

Sometimes it is helpful (or a maintainer asks for it) to test your library changes in the final binary, e.g. podman.

Assume we like to test a containers/common PR in Podman so that we can have the full CI tests run there.
First you need to push your containers/common changes to your github fork (if not already done).
Now open the podman repository, create a new branch there and then use.
```
$ go mod edit -replace github.com/containers/common=github.com/<account name>/<fork name>@<branch name>
```
Replace the variable with the correct values, in my case it the reference might be `github.com/Luap99/common@netns-dir`, where
 - account name == `Luap99`
 - fork name == `common`
 - branch name that I like to test == `netns-dir`

Then just run the vendor command again.
```
$ make vendor
```

Now do any other changes that might be needed after the update and commit the changes then push them
to your Podman fork and open a new Podman PR, marking it as draft to make clear that this is a test
and should not be merged. This will trigger CI to run the tests. If everything passes the
containers/common PR did not introduce any regression which is a good.

Note: You generally do not have to test all your library changes like that. However if your changes
are big or break the API it might be a good idea to do do this to avoid regression that need to be
fixed in follow ups or revert.

## git bisect a change in a go dependency

If you performed a the git bisect and the resulting commit is one that updated a library then most likely
the problem is in that library instead. In such cases it may be needed to find the bad commit from this
repository instead. Thankfully this is not much more difficult than the normal bisect usage.

Clone the library repository locally (for this example assume we it is github.com/containers/storage),
I assume it is in a directory next to the podman repo.

Then in podman run (where you replace the path to the storage repo with your actual one)
```
$ go mod edit -replace github.com/containers/storage=/path/to/storage
$ make vendor
```

Now the commit that was already found via the bisect in Podman should show you which storage version
was changed so you can then use them as good and bad version for the bisect in storage.

So use them in the storage repo for the `git bisect start BAD GOOD` command and then we need a bit
more work for the testing as we have to compile podman in the other repo and perform the check there.

The automated command can look like this:
```
$ git bisect run sh -c "cd /path/to/podman && make vendor && make podman && podman run $IMAGE someCommand || exit 1"
```

Compared to the normal bisect we basically just have to switch to the podman repo and then update
the vendor directory, as this will copy the local storage repo into that so the build after it
gets the current changes from the bisect commit. Given all works fine the result will point you
to a single commit in storage that caused the podman problem.
