## VARS

###  Fuzz

`go test --fuzz ^FuzzVariableKeys$`

###  Benchmark

## Variable key

Purpose of this benchmark is to ensure that we actually have performance gain parsing variable keys compared to using standard libraries.

`go test -race -timeout 30s -benchmem -bench ^BenchmarkParseKeys$ .`

| | | | | |
| --- | --- | --- | --- | --- |
| **single valid key** | | | | |
| parseKey_common_key-8 | 61692270 | 19.75 ns/op | 0 B/op | 0 allocs/op |
| parseKeyStd_common_key-8 | 15147985 | 80.61 ns/op | 0 B/op | 0 allocs/op |
| **set of valid keys** | | | | |
| parseKey_valid_keys(44)-8 | 553406 | 2136 ns/op | 0 B/op | 0 allocs/op |
| parseKeyStd_valid_keys(44)-8 | 228105 | 4530 ns/op | 0 B/op | 0 allocs/op |
| **set of bad keys** | | | | |
| parseKey_bad_keys(37)-8 | 1782696 | 678.7 ns/op | 0 B/op | 0 allocs/op |
| parseKeyStd_bad_keys(37)-8 | 811699 | 1475 ns/op | 0 B/op | 0 allocs/op |

## Variable maps

Purpose of this benchmark is to ensure that we actually have performance gain on vars.Map compared to using sync.Map.


**Single variable and perform basic rw ops** 

`go test -race -timeout 30s -benchmem -bench ^BenchmarkVariableMapSingleValue$ .`

| | | | | |
| --- | --- | --- | --- | --- |
| vars.Map-8 | 218181 | 8911 ns/op | 936 B/op | 5 allocs/op |
| sync.Map-8 | 13838 | 5940 ns/op | 408 B/op | 10 allocs/op |


**Set of variables and perform basic rw ops**

`go test -race -timeout 30s -benchmem -bench ^BenchmarkVariableMaps$ .`

| | | | | |
| --- | --- | --- | --- | --- |
| vars.Map-8 | 14251 | 78694 ns/op | 3083 B/op | 18 allocs/op |
| sync.Map-8 | 13838 | 81988 ns/op | 2712 B/op | 68 allocs/op |
