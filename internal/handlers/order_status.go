package handlers

import "strings"

const (
	OrderStatusPending    = "Pending"
	OrderStatusInProgress = "In Progress"
	OrderStatusProcessing = "Processing"
	OrderStatusShipped    = "Shipped"
	OrderStatusTransit    = "Transit"
	OrderStatusDelivered  = "Delivered"
	OrderStatusCancelled  = "Cancelled"
)

var orderStatuses = []string{
	OrderStatusPending,
	OrderStatusInProgress,
	OrderStatusProcessing,
	OrderStatusShipped,
	OrderStatusTransit,
	OrderStatusDelivered,
	OrderStatusCancelled,
}

var orderStatusAliases = map[string]string{
	"pending":      OrderStatusPending,
	"in progress":  OrderStatusInProgress,
	"processing":   OrderStatusProcessing,
	"shipped":      OrderStatusShipped,
	"transit":      OrderStatusTransit,
	"delivered":    OrderStatusDelivered,
	"cancelled":    OrderStatusCancelled,
	"canceled":     OrderStatusCancelled,
}

func normalizeOrderStatus(input string) (string, bool) {
	normalized := strings.TrimSpace(strings.ToLower(input))
	if normalized == "" {
		return "", false
	}
	normalized = strings.ReplaceAll(normalized, "_", " ")
	normalized = strings.ReplaceAll(normalized, "-", " ")
	normalized = strings.Join(strings.Fields(normalized), " ")
	status, ok := orderStatusAliases[normalized]
	return status, ok
}

func isTerminalOrderStatus(status string) bool {
	normalized, ok := normalizeOrderStatus(status)
	if !ok {
		return false
	}
	return normalized == OrderStatusCancelled || normalized == OrderStatusDelivered
}
