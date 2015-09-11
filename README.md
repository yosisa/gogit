# gogit
A pure Go Git library.

This package provides set of tools to handle a git repository directly from Go program without external `git` command.

Highlighted features are:

* Handle a git repository including a bare repository.
* Get a commit, tree, blob or tag object from a repository.
* Parse pack files and pack index v2 files (pack index v1 not yet supported).
* Parse `packed-refs` file.
* Objects and refs are seamlessly resolved whether it's packed or not.
* Implemented by only Go, no need for cgo or external `git` command.

Currently, it supports read access only. But supporting write access is planed.

This is just worked but in early development stage, breaking changes maybe introduced suddenly. So please consider to use vendoring.

## License
MIT
