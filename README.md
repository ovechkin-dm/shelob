# Shelob

## Simple web crawler

### Installation

```shell
go install github.com/ovechkin-dm/shelob@latest
```

### Example usage

```shell
shelob --baseurl=https://google.com --debug --downloadpath="./downloads/" --resume --workers=10
```

### Options

```
Usage of ./shelob:
      --baseurl string        Base URL for the crawler
      --debug                 Enable debug mode
      --downloadpath string   Base path to download content (default "./downloads/")
      --resume                Resume previous download
      --workers int           Number of workers to use (default 1)
```