![Happy Logo](assets/images/happy.svg)

# Howijd Prototyping Framework and SDK

Package happy is a powerful tool for developers looking to bring their ideas to life through rapid prototyping. With its comprehensive set of resources and modular design, it's easy to create working prototypes or MVPs with minimal technical knowledge or infrastructure planning. Plus, its flexible design allows it to seamlessly integrate into projects with components written in different programming languages. So why wait? Let Happy help you achieve your goals and bring a smile to your face along the way.

:warning: *Happy is very early in development phase and is not intended for production use.*  

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

## Credits

[![GitHub contributors](https://img.shields.io/github/contributors/mkungla/happy?style=flat-square)](https://github.com/mkungla/happy/graphs/contributors)

<sub>**Happy banner design.**</sub>  
<sup>Happy banner was designed by Egon Elbre <a href="https://egonelbre.com/" target="_blank">egonelbre.com</a></sup>
