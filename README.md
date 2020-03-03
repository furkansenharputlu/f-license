# f-license
> **NOTE**: This project will be improved with lots of new features!

**f-license** is an open-source license activation tool.

# How to use

## Start f-license server

1. Create and configure `config.json` file like `sample_config.json`
2. Run `go build`
3. Run `./f-license` 

## Embed client code to your example app

In the **example** directory, you can access to a simple usage of the activation in your Go application. There are two variables you need to set:

- `license`: a license created by **f-license** 
- `secret`: a private secret to validate the license

## Generate license

**Sample request**

```
POST /admin/generate HTTP/1.1
Host: localhost:4242
Authorization: admin123
```

**Sample response**

```
{
    "license": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.bnVsbA.BF4LHTey6WwVbYuKSP_kyfeB2-PZPEJYHmkdp_R92y4",
    "license_hash": 12661635182986732340
}
```

## Activate license

**Sample request**

```
PUT /admin/activate HTTP/1.1
Host: localhost:4242
Authorization: admin123
Content-Type: application/x-www-form-urlencoded

license_hash=12661635182986732340
```

**Sample response**

```
{
    "message": "Activated"
}
```

## Inactivate license

**Sample request**

```
PUT /admin/inactivate HTTP/1.1
Host: localhost:4242
Authorization: admin123
Content-Type: application/x-www-form-urlencoded

license_hash=12661635182986732340
```

**Sample response**

```
{
    "message": "Inactivated"
}
```
