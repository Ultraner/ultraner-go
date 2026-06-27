# Ultraner Go SDK

One API for payments across Africa: mobile money, cards, PayPal and wallets. Live in Tanzania and Rwanda, expanding across the continent.

- Docs: https://ultraner.com/docs
- OpenAPI: https://ultraner.com/openapi.json
- For AI: https://ultraner.com/ai

## Install

```bash
go get github.com/ultraner/ultraner-go
```

## Usage

```go
package main

import (
	"context"
	"fmt"

	ultraner "github.com/ultraner/ultraner-go"
)

func main() {
	client := ultraner.New("sk_live_...")
	ctx := context.Background()

	payment, err := client.CreateMobileMoney(ctx, ultraner.MobileMoneyCharge{
		Amount:        5000,
		Currency:      "TZS",
		Provider:      "Vodacom",
		AccountNumber: "255700000000",
		ExternalID:    "order_1001",
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(payment.Reference, payment.Status)

	status, _ := client.PaymentStatus(ctx, payment.Reference)
	fmt.Println(status.Status)

	_, _ = client.Wallet(ctx)
	_, _ = client.Transactions(ctx, 1, 20)
}
```

Errors are returned as `*ultraner.Error` with `Status` and `Code`.

## License

MIT
