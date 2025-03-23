# Build

```
docker build -t devcontainer .
```

# Use

```
docker run -it -v "/var/run/docker.sock:/var/run/docker.sock" -v "$(pwd):/workspace" devcontainer ...
```