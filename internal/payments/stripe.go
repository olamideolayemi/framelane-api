package payments

import (
  "context"
  "github.com/stripe/stripe-go/v79"
  "github.com/stripe/stripe-go/v79/paymentintent"
)

type Stripe struct{}

func NewStripe(secret string) *Stripe {
  stripe.Key = secret
  return &Stripe{}
}

func (s *Stripe) CreateIntent(ctx context.Context, amount int64, currency, customerEmail string, metadata map[string]string) (*stripe.PaymentIntent, error) {
  params := &stripe.PaymentIntentParams{
    Amount:   stripe.Int64(amount),
    Currency: stripe.String(currency),
    AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
      Enabled: stripe.Bool(true),
    },
    ReceiptEmail: stripe.String(customerEmail),
  }
  for k, v := range metadata {
    params.AddMetadata(k, v)
  }
  return paymentintent.New(params)
}
