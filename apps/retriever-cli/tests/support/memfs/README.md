# memfs

An in-memory file tree for the unit tests. It holds regular files and folders under absolute slash paths, and it offers
the calls that the catalog and env-file loaders make: `Lstat`, `Open`, and `SameFile`. A unit test passes these methods
into `catalog.NewWithFiles` and `envfile.NewWithFiles`, so the loader reads the tree and no disk.

The tree never touches the disk, the process environment, or a child process. Its own tests are untagged, and they pass
the untagged rule in `tests/architecture`.
