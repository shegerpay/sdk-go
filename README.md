<p align="center"><img src="logo.png" alt="ShegerPay" width="200" /></p>

# ShegerPay Go SDK

[![Version](https://img.shields.io/badge/version-2.2.0-blue)](https://pkg.go.dev/github.com/shegerpay/sdk-go)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

Official Go SDK for ShegerPay — verify Ethiopian bank payments (CBE, Telebirr, BOA, Awash).

## Install

```bash
go get github.com/shegerpay/sdk-go
```

## Quick Start

```go
package main

import (
    "fmt"
    shegerpay "github.com/shegerpay/sdk-go"
)

func main() {
    client := shegerpay.NewClient("sk_live_YOUR_API_KEY")

    // Verify a payment
    result, err := client.Verify("FT26062K7WMY", 1000, "cbe", nil)
    if err == nil && result.Verified {
        fmt.Println("Payment verified!")
    }

    // Verify without amount (lookup only)
    result2, _ := client.Verify("FT26062K7WMY", 0, "telebirr", nil)
    fmt.Println(result2.Status)

    // Verify from receipt screenshot
    imageResult, _ := client.VerifyImage("base64_image_string", "cbe", nil)
    fmt.Println(imageResult.Verified)

    // Create payment link
    link, _ := client.CreatePaymentLink(map[string]interface{}{
        "title":    "Order #1234",
        "amount":   1500,
        "currency": "ETB",
    })
    fmt.Println(link["url"])
}
```

## Supported Providers
`cbe` · `telebirr` · `boa` · `awash` · `ebirr_kaafi` · `ebirr_coop`

## Requirements
- Go 1.21+


## Support
- 📚 Docs: https://shegerpay.com/docs
- 💬 Telegram: [@shegerpay_0](https://t.me/shegerpay_0)
- 📧 Email: support@shegerpay.com
