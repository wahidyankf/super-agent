# Shared libraries

`libs/` will hold code that more than one application uses. Nothing is here yet.

Libraries will sit flat: one directory for each library, directly under `libs/`. The root `package.json` already
lists `libs/*` as a workspace pattern. Add a library only when a second application needs the same code.
