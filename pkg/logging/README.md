# Logging

## Testing

**Run all tests in logging package using one of the following commands**

```
go test ./... 
go test -race ./... 
go test -race -coverprofile=coverage.out ./... 
go test -race -coverprofile=coverage.out ./... && go tool cover -html=coverage.out
```

### Benchmarks

**Run all Benchmarks**

```
go test -bench=. -benchmem
```
