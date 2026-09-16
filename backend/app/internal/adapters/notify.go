package adapters

import (
	"context"
	"fmt"
	"time"

	"undangan/kernel/authctx"
	billingdomain "undangan/services/billing/domain"
	billingservice "undangan/services/billing/service"
	notifydomain "undangan/services/notify/domain"
	userdomain "undangan/services/user/domain"
)

// TelegramOrders memenuhi notify/domain.OrderOps memakai service billing.
// Svc diisi setelah konstruksi karena billing & bot saling membutuhkan (bot memanggil approve/reject,
// billing mengirim notifikasi lewat bot).
type TelegramOrders struct{ Svc billingservice.OrderService }

var _ notifydomain.OrderOps = (*TelegramOrders)(nil)

func (a *TelegramOrders) Approve(ctx context.Context, actor authctx.Principal, orderID string) (string, error) {
	o, err := a.Svc.Approve(ctx, actor, orderID)
	if err != nil {
		return "", err
	}
	return o.Code, nil
}

func (a *TelegramOrders) Reject(ctx context.Context, actor authctx.Principal, orderID, reason string) (string, error) {
	o, err := a.Svc.Reject(ctx, actor, orderID, reason)
	if err != nil {
		return "", err
	}
	return o.Code, nil
}

func (a *TelegramOrders) Pending(ctx context.Context) ([]notifydomain.PendingOrder, error) {
	items, _, err := a.Svc.List(ctx, billingdomain.OrderFilter{Status: string(billingdomain.OrderAwaitingConfirmation), Limit: 20})
	if err != nil {
		return nil, err
	}
	out := make([]notifydomain.PendingOrder, 0, len(items))
	for _, o := range items {
		out = append(out, notifydomain.PendingOrder{
			ID: o.ID, Code: o.Code, Customer: o.UserName, PlanName: o.PlanName, Amount: o.Amount,
			Status: string(o.Status), HasProof: o.ProofPath != "", CreatedAgo: ago(o.CreatedAt),
		})
	}
	return out, nil
}

func ago(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "baru saja"
	case d < time.Hour:
		return fmt.Sprintf("%d menit lalu", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%d jam lalu", int(d.Hours()))
	default:
		return fmt.Sprintf("%d hari lalu", int(d.Hours()/24))
	}
}

// SystemAdmin memenuhi notify/domain.AdminResolver: super admin terlama dipakai sebagai pelaku aksi bot.
type SystemAdmin struct{ Repo userdomain.Repository }

var _ notifydomain.AdminResolver = SystemAdmin{}

func (a SystemAdmin) SystemAdmin(ctx context.Context) (authctx.Principal, error) {
	users, _, err := a.Repo.List(ctx, userdomain.ListFilter{Role: authctx.RoleSuperAdmin, Limit: 1})
	if err != nil {
		return authctx.Principal{}, err
	}
	if len(users) == 0 {
		return authctx.Principal{}, fmt.Errorf("super admin belum ada")
	}
	u := users[0]
	return authctx.Principal{UserID: u.ID, Role: u.Role, Name: u.Name, Email: u.Email}, nil
}
