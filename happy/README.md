## examples

**hello**

*most minimal usage*

```
go run ./examples/hello/
go run ./examples/hello/ nickname
# increase verbosity
go run ./examples/hello/ --debug
go run ./examples/hello/ --system-debug
# help
go run ./examples/hello/ -h
```

**kitchensink**

*main application when no subcommand is provided*

```
go run ./examples/kitchensink/
# increase verbosity
go run ./examples/kitchensink/ --verbose
go run ./examples/kitchensink/ --debug
go run ./examples/kitchensink/ --system-debug

# main application help
go run ./examples/kitchensink/ -h
```

*hello command with flags*

```
go run ./examples/kitchensink/ hello --name me --repeat 10 
# or shorter
go run ./examples/kitchensink/ hello -n me -r 10 

# help for hello command
go run ./examples/kitchensink/ hello -h
```
