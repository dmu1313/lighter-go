
This directory holds transaction signing code that is essentially a copy of the `sharedlib/` directory in this repo. However, the `sharedlib/` directory utilizes cgo whereas this directory is solely meant to be consumed by other Go programs.

Some of the code will appear pointless. For example, parameters of a certain type such as int64 may be typecast to int64, which is redundant. The goal of this rewrite is not to improve upon the existing signing code but rather to copy it as closely as possible such that it's extremely easy to update this package when the upstream repo is changed.
