# Annotated Go code

```go
func main() {
    config := loadConfig()
    server := newServer(config)
    server.Run()
}
```

{code highlight="2-3" note="2:Load configuration." note="3:Create the server." reference-position="right" reference-color="yellow"}
