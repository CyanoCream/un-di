// Package domain (dashboard) = read model statistik admin (lintas tabel, tanpa entity yang diubah).
package domain

import "context"

type Stats struct {
	Users                int   `json:"users"`
	ActiveSubscriptions  int   `json:"active_subscriptions"`
	GraceSubscriptions   int   `json:"grace_subscriptions"`
	PendingOrders        int   `json:"pending_orders"`
	Invitations          int   `json:"invitations"`
	PublishedInvitations int   `json:"published_invitations"`
	RevenueThisMonth     int64 `json:"revenue_this_month"`
	Guests               int   `json:"guests"`
}

type Repository interface {
	Stats(ctx context.Context) (*Stats, error)
}
