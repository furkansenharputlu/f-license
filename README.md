# f-license

**f-license** is a tool to use in your apps for license activation.

# How to use

## Start f-license server

1. Create and configure `config.json` file like `sample_config.json`
2. Run `go build`
3. Run `./f-license` 


## Embed client code to your example app

In the **example** directory, you can access to a simple usage of the activation in your Go application. There are two variables you need to set:

- `license`: a license created by **f-license** 
- `secret`: a private secret to validate the license

> NOTE: This project will be improved with lots of new features!
