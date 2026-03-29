# CLAUDE.md - Realmicro Project Guide

## Project Overview

Realmicro is a framework for distributed systems development in Go. It provides pluggable abstractions for service discovery, RPC, pub/sub, config, auth, storage, and more.

The framework is evolving into an **AI-native platform** where every microservice is automatically accessible to AI agents via the Model Context Protocol (MCP).

## Project Structure

```
go-micro/
├── ai/             # AI model providers (Anthropic, OpenAI)
├── auth/           # Authentication (JWT, no-op)
├── broker/         # Message broker (NATS, RabbitMQ)
├── cache/          # Caching (Redis)
├── client/         # RPC client (gRPC)
├── codec/          # Message codecs (JSON, Proto)
├── config/         # Dynamic config (env, file, etcd, NATS)
├── errors/         # Error handling
├── logger/         # Logging
├── metadata/       # Context metadata
├── registry/       # Service discovery (mDNS, Consul, etcd)
├── selector/       # Client-side load balancing
├── server/         # RPC server
├── service/        # Service interface + profiles
├── store/          # Data persistence (Postgres, NATS KV)
├── transport/      # Network transport
├── wrapper/        # Middleware (auth, trace, metrics)
├── examples/       # Working examples
└── common/         # Non-public: docs, utils, test harness
```

## 详细规范

编码风格、架构模式、Proto 规范、测试规范、Git 工作流等详见 `.claude/rules/` 目录下的规则文件。