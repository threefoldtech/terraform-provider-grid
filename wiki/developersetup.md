## Developer setup

- should be using vscode
- make sure to have `git`, `make`, `taskfile` installed
- `make submodules` to populate the git submodules in the project


## Before committing code
- Make sure to run the using `make checks`
- Make sure tests pass using `make unittests`

## Integration tests
Integration tets happen on the repository on the PRs

> TO run all tests `make tests`

## Go and Code reviews
- [Effective Go](https://go.dev/doc/effective_go)
- [CodeReview Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Common mistakes](https://github.com/golang/go/wiki/CommonMistakes)
- Any code review guide works, recommending [uber's go guide](https://github.com/uber-go/guide)


## Generating terraform documentation
should happen by using `make docs`