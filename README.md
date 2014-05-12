tonic
-----
A service reloading tool for use in development.

Built with bits and pieces from gin (github.com/codegangsta/gin).

Tonic will scan your current dir and rebuild your daemon on any change
to a `.go` file. Very crude, very simple.

```
 $ # to install:
 $ go get github.com/stengaard/tonic
 $ # to use in development:
 $ cd $GOPATH/go/src/myapp
 $ tonic run
 [tonic] Build successful
 [myapp] ready
```
