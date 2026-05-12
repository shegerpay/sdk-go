# ShegerPay Go SDK

[![Version](https://img.shields.io/badge/version-2.2.0-blue)](https://pkg.go.dev/github.com/shegerpay/sdk-go)

Official Go SDK for [ShegerPay](https://shegerpay.com) — Ethiopian payment verification.

## Install

```sh
go get github.com/shegerpay/sdk-go
```

## Quick Start

```go
client := shegerpay.NewClient("sk_live_YOUR_API_KEY")
result, _ := client.Verify("FT26062K7WMY", 1000, "cbe", nil)
fmt.Println(result.Verified)
```

## Docs

https://shegerpay.com/docs
