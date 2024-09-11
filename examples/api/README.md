# API Gateway

An api gateway for RealMicro services

## Overview

The API gateway dynamically serves requests via HTTP json to realmicro services using path based resolution.

## Example

The request http://localhost:8080/helloworld/call will route to the service `realmicro.helloworld` and endpoint `Helloworld.Call`.

### For differing handlers

The request http://localhost:8080/helloworld/Greeter/Hello will route to the service `realmicro.helloworld` and endpoint `Greeter.Hello`

## Usage

Install

```
go install github.com/realmicro/realmicro/example/api
```

Run (listens on :9090)

```
api
```

To test it in Postman, create `Post` request with url `http://localhost:8080/helloworld/Greeter/Call`, use `x-www-form-urlencoded` format, that's all. Notice: latest code in realmicro master branch supports this way, if not it will report a `ill-formed: POST` error.

Test with curl:
```shell
curl -L http://localhost:9090/helloworld/Greeter/Hello -X POST -d '{"name": "world"}' --header "Content-Type: application/json"
```

## TODO

- Enable changing registry, client/server, etc
- Enable setting endpoints manually via RPC