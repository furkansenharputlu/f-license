# f-license
> **NOTE**: This project will be improved with lots of new features!

**f-license** is an open-source license creation and verification tool. You can quickly add license key verification to your application. Don't implement yourself, just use the open-source product!

# Features

- Generating license key with a private secret
- Remote verification of a license key
- Local verification of a license key
- Storing licence keys in MongoDB
- Activating and inactivating customer license keys remotely
- **f-cli** tool to manage licenses by terminal

See the latest [Documentation](https://github.com/furkansenharputlu/f-license/wiki).

# How to use

## Prerequisites

- MongoDB server

## Start f-license server

1. Create and configure `config.json` file like [sample_config.json](https://github.com/furkansenharputlu/f-license/blob/master/sample_config.json)
2. Run `go build`
3. Run `./f-license` 

## Embed client code to your app

If your app's language is `Go`, you need to add just one line code to your application after importing `fclient`.

```go
import "github.com/furkansenharputlu/f-license/fclient"
```

### Remote verification

```go
verified, err := fclient.VerifyRemotely("http://localhost:4242", "license-key")
```

### Local verification

```go
verified, err := fclient.VerifyLocally("secret-or-public-key", "license-key")
```

If you are not using `Go`, you can easily implement their equivalent in your app's languge for now. In future, we will implement for different languages.

## CLI usage

1. Run `go build -o f-cli ./cli`
2. Generate `license.json` like [sample_license.json](https://github.com/furkansenharputlu/f-license/blob/master/sample_license.json)
3. Run `./f-cli generate license.json`
4. Run `./f-cli verify <license-token>`
5. Run `./f-cli -h` to see other commands
