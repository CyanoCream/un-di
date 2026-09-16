// Package controller (billing) = HTTP handler paket, order, langganan, dan pengaturan pembayaran.
package controller

import (
	"net/http"

	"undangan/kernel/authctx"
	"undangan/kernel/httpx"
	"undangan/kernel/storage"
	"undangan/services/billing/domain"
	"undangan/services/billing/service"
)

type Controller struct {
	plans    service.PlanService
	orders   service.OrderService
	subs     service.SubscriptionService
	settings service.SettingsService
}

func New(plans service.PlanService, orders service.OrderService, subs service.SubscriptionService, settings service.SettingsService) *Controller {
	return &Controller{plans: plans, orders: orders, subs: subs, settings: settings}
}

func (c *Controller) Register(rt *httpx.Router) {
	rt.Auth("GET /api/v1/plans", c.listActivePlans)
	rt.Auth("GET /api/v1/payment-settings", c.getPaymentSettings)
	rt.Auth("GET /api/v1/orders/{id}/proof", c.proof)

	rt.Customer("GET /api/v1/me/subscription", c.mySubscription)
	rt.Customer("GET /api/v1/me/orders", c.myOrders)
	rt.Customer("POST /api/v1/me/orders", c.createOrder)
	rt.Customer("GET /api/v1/me/orders/{id}", c.getOrder)
	rt.Customer("POST /api/v1/me/orders/{id}/proof", c.uploadProof)

	rt.Admin("GET /api/v1/admin/plans", c.listPlans)
	rt.Admin("POST /api/v1/admin/plans", c.createPlan)
	rt.Admin("PATCH /api/v1/admin/plans/{id}", c.updatePlan)
	rt.Admin("GET /api/v1/admin/orders", c.listOrders)
	rt.Admin("GET /api/v1/admin/orders/{id}", c.getOrder)
	rt.Admin("POST /api/v1/admin/orders/{id}/approve", c.approve)
	rt.Admin("POST /api/v1/admin/orders/{id}/reject", c.reject)
	rt.Admin("GET /api/v1/admin/subscriptions", c.listSubscriptions)
	rt.Admin("POST /api/v1/admin/users/{id}/subscriptions", c.grant)
	rt.Admin("GET /api/v1/admin/settings/payment", c.getPaymentSettings)
	rt.Admin("PUT /api/v1/admin/settings/payment", c.updatePaymentSettings)
}

// ---- paket ----

func (c *Controller) listActivePlans(w http.ResponseWriter, r *http.Request) error {
	items, err := c.plans.ListActive(r.Context())
	if err != nil {
		return err
	}
	httpx.OK(w, httpx.NewList(items))
	return nil
}

func (c *Controller) listPlans(w http.ResponseWriter, r *http.Request) error {
	items, err := c.plans.ListAll(r.Context())
	if err != nil {
		return err
	}
	httpx.OK(w, httpx.NewList(items))
	return nil
}

func (c *Controller) createPlan(w http.ResponseWriter, r *http.Request) error {
	var in service.PlanInput
	if err := httpx.Decode(r, &in); err != nil {
		return err
	}
	p, err := c.plans.Create(r.Context(), authctx.MustFrom(r.Context()), in)
	if err != nil {
		return err
	}
	httpx.Created(w, p)
	return nil
}

func (c *Controller) updatePlan(w http.ResponseWriter, r *http.Request) error {
	var in service.PlanInput
	if err := httpx.Decode(r, &in); err != nil {
		return err
	}
	p, err := c.plans.Update(r.Context(), authctx.MustFrom(r.Context()), r.PathValue("id"), in)
	if err != nil {
		return err
	}
	httpx.OK(w, p)
	return nil
}

// ---- order ----

func (c *Controller) myOrders(w http.ResponseWriter, r *http.Request) error {
	p := httpx.PageFrom(r)
	q := r.URL.Query()
	items, total, err := c.orders.List(r.Context(), domain.OrderFilter{
		UserID: authctx.MustFrom(r.Context()).UserID, Status: q.Get("status"), Limit: p.PerPage, Offset: p.Offset(),
	})
	if err != nil {
		return err
	}
	httpx.OK(w, httpx.NewPaginated(items, total, p))
	return nil
}

func (c *Controller) listOrders(w http.ResponseWriter, r *http.Request) error {
	p := httpx.PageFrom(r)
	q := r.URL.Query()
	items, total, err := c.orders.List(r.Context(), domain.OrderFilter{
		Status: q.Get("status"), Query: q.Get("q"), Limit: p.PerPage, Offset: p.Offset(),
	})
	if err != nil {
		return err
	}
	httpx.OK(w, httpx.NewPaginated(items, total, p))
	return nil
}

func (c *Controller) createOrder(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		PlanID string `json:"plan_id"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		return err
	}
	o, err := c.orders.Create(r.Context(), authctx.MustFrom(r.Context()), in.PlanID)
	if err != nil {
		return err
	}
	httpx.Created(w, o)
	return nil
}

// getOrder dipakai customer (hanya miliknya, dicek service) dan admin.
func (c *Controller) getOrder(w http.ResponseWriter, r *http.Request) error {
	o, err := c.orders.Get(r.Context(), authctx.MustFrom(r.Context()), r.PathValue("id"))
	if err != nil {
		return err
	}
	httpx.OK(w, o)
	return nil
}

func (c *Controller) uploadProof(w http.ResponseWriter, r *http.Request) error {
	if err := httpx.ParseMultipart(w, r, storage.ImageRule.MaxBytes); err != nil {
		return err
	}
	data, _, err := httpx.FormFile(r, storage.ImageRule.MaxBytes)
	if err != nil {
		return err
	}
	o, err := c.orders.UploadProof(r.Context(), authctx.MustFrom(r.Context()), r.PathValue("id"), data)
	if err != nil {
		return err
	}
	httpx.OK(w, o)
	return nil
}

func (c *Controller) proof(w http.ResponseWriter, r *http.Request) error {
	f, err := c.orders.OpenProof(r.Context(), authctx.MustFrom(r.Context()), r.PathValue("id"))
	if err != nil {
		return err
	}
	defer f.Body.Close()
	h := w.Header()
	h.Set("Content-Type", f.ContentType)
	h.Set("Cache-Control", "private, no-store")
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Content-Disposition", `inline; filename="`+f.Name+`"`)
	http.ServeContent(w, r, f.Name, f.ModTime, f.Body)
	return nil
}

func (c *Controller) approve(w http.ResponseWriter, r *http.Request) error {
	o, err := c.orders.Approve(r.Context(), authctx.MustFrom(r.Context()), r.PathValue("id"))
	if err != nil {
		return err
	}
	httpx.OK(w, o)
	return nil
}

func (c *Controller) reject(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Reason string `json:"reason"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		return err
	}
	o, err := c.orders.Reject(r.Context(), authctx.MustFrom(r.Context()), r.PathValue("id"), in.Reason)
	if err != nil {
		return err
	}
	httpx.OK(w, o)
	return nil
}

// ---- langganan ----

func (c *Controller) mySubscription(w http.ResponseWriter, r *http.Request) error {
	out, err := c.subs.ForUser(r.Context(), authctx.MustFrom(r.Context()).UserID)
	if err != nil {
		return err
	}
	httpx.OK(w, out)
	return nil
}

func (c *Controller) listSubscriptions(w http.ResponseWriter, r *http.Request) error {
	p := httpx.PageFrom(r)
	q := r.URL.Query()
	items, total, err := c.subs.List(r.Context(), domain.SubscriptionFilter{
		Status: q.Get("status"), Query: q.Get("q"), Limit: p.PerPage, Offset: p.Offset(),
	})
	if err != nil {
		return err
	}
	httpx.OK(w, httpx.NewPaginated(items, total, p))
	return nil
}

func (c *Controller) grant(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		PlanID string `json:"plan_id"`
		Days   int    `json:"days"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		return err
	}
	sub, err := c.subs.GrantByAdmin(r.Context(), authctx.MustFrom(r.Context()), r.PathValue("id"), in.PlanID, in.Days)
	if err != nil {
		return err
	}
	httpx.Created(w, sub)
	return nil
}

// ---- pengaturan pembayaran ----

func (c *Controller) getPaymentSettings(w http.ResponseWriter, r *http.Request) error {
	s, err := c.settings.Payment(r.Context())
	if err != nil {
		return err
	}
	httpx.OK(w, s)
	return nil
}

func (c *Controller) updatePaymentSettings(w http.ResponseWriter, r *http.Request) error {
	var in domain.PaymentSettings
	if err := httpx.Decode(r, &in); err != nil {
		return err
	}
	s, err := c.settings.UpdatePayment(r.Context(), authctx.MustFrom(r.Context()), in)
	if err != nil {
		return err
	}
	httpx.OK(w, s)
	return nil
}
