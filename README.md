# f-license
> **NOTE**: This project will be improved with lots of new features!

**f-license** is an open-source license creation and verification tool. You can quickly add license key verification to your application. Don't implement yourself, just use the open-source product!

# Features

- Generating license key with a private secret
- Remote verification of a license key
- Local verification of a license key
- Storing licence keys in MongoDB
- Activating and inactivating customer license keys remotely

# How to use

## Prerequisites

- MongoDB server

## Start f-license server

1. Create and configure `config.json` file like `sample_config.json`
2. Run `go build`
3. Run `./f-license` 

## Embed client code to your example app

In the **example** directory, you can access to a simple usage of the activation in your Go application. There are two variables you need to set:

- `license`: a license created by **f-license** 
- `secret`: a private secret to validate the license

## Documentation

https://github.com/furkansenharputlu/f-license/wiki/Documentation
