# Service Ping Pong example

Illustrates service communication.

3 peer services send messages to each other  
and the monitoring service intercepts these messages.  
The monitoring service closes services and app when at least  
`--messages n' messages are intercepted.

```shell
go run . --help
go run . start --help
go run . start
go run . start --messages 10 --json pretty
go run . start --messages 10000
```
