A fork of Golang library for generating random strings from regular expressions.

I took https://github.com/zach-klippenstein/goregen and made some changes to make it more suitable for my use:

* use crypto.rand (by default and always)
* remove use of all testing dependencies (most of them not all that useful)
* add go.mod (which is empty though)
