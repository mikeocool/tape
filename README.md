A cli tool for managing devcontainers defined by some simple yaml config.

Setup config:
```
mkdir .tape
cp fixtures/.tape/hellobox.yml .tape/
```

Build: 
```
mkdir bin
go build -o bin/tape .
```

```
./bin/tape ls
./bin/tape up hellobox
./bin/tape exec hellobox ls -- -al
```

Run tests
```
go test ./...
```

TODO
issue running multiple dev containers with same workspace