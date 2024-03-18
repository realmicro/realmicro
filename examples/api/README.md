# API Gateway

An api gateway for RealMicro services

## Overview

The API gateway dynamically serves requests via HTTP json to realmicro services using path based resolution.

## Example

The request http://localhost:8080/helloworld/call will route to the service `real.micro.helloworld` and endpoint `Helloworld.Call`.

### For differing handlers

The request http://localhost:8080/helloworld/Greeter/Call will route to the service `real.micro.helloworld` and endpoint `Greeter.Call`

## Usage

Install

```
go install github.com/realmicro/realmicro/example/api
```

Run (listens on :8080)

```
api
```

To test it in Postman, create `Post` request with url `http://localhost:8080/helloworld/Greeter/Call`, use `x-www-form-urlencoded` format, that's all. Notice: latest code in realmicro master branch supports this way, if not it will report a `ill-formed: POST` error.

## TODO

- Enable changing registry, client/server, etc
- Enable setting endpoints manually via RPC