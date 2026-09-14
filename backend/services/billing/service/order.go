package service

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"math/rand/v2"
	"path"
	"strings"
	"time"

	auditport "undangan/kernel/audit"
	"undangan/services/billing/domain"
	"undangan/kernel/apperror"
	"undangan/kernel/database"
	"undangan/kernel/security"
	"undangan/kernel/storage"
	"undangan/kernel/authctx"
)

// ProofFile = bukti bayar yang siap di-stream; pemanggil wajib menutup Body.
type ProofFile struct {
	Body        io.ReadSeekCloser
	ModTime     time.Time
	ContentType string
	Name        string
}

type OrderService interface {
	// Create order untuk paket aktif (customer).
	Create(ctx context.Context, actor authctx.Principal, planID string) (*domain.Order, error)
	// Get: customer hanya order miliknya (selain itu 404); super admin semua.
	Get(ctx context.Context, actor authctx.Principal, id string) (*domain.Order, error)
	List(ctx context.Context, f domain.OrderFilter) ([]domain.Order, int, error)
	UploadProof(ctx context.Context, actor authctx.Principal, id string, data []byte) (*domain.Order, error)
	OpenProof(ctx context.Context, actor authctx.Principal, id string) (*ProofFile, error)
	Approve(ctx context.Context, actor authctx.Principal, id string) (*domain.Order, error)
	Reject(ctx context.Context, actor authctx.Principal, id, reason string) (*domain.Order, error)
}

type orderService struct {
	orders domain.OrderRepository
	plans  domain.PlanRepository
	subs   SubscriptionService
	files  storage.FileStorage
	locker domain.Locker
	audit  auditport.Recorder
	tx     database.TxManager
	now    func() time.Time
}

func NewOrderService(orders domain.OrderRepository, plans domain.PlanRepository, subs SubscriptionService, files storage.FileStorage,
	locker domain.Locker, audit auditport.Recorder, tx database.TxManager) OrderService {
	return &orderService{orders: orders, plans: plans, subs: subs, files: files, locker: locker, audit: audit, tx: tx, now: time.Now}
}

var errOrderNotFound = apperror.NotFound("Order tidak ditemukan")

func (s *orderService) find(ctx context.Context, id string, forUpdate bool) (*domain.Order, error) {
	var o *domain.Order
	var err error
	if forUpdate {
		o, err = s.orders.FindByIDForUpdate(ctx, id)
	} else {
		o, err = s.orders.FindByID(ctx, id)
	}
	if errors.Is(err, apperror.ErrNotFound) {
		return nil, errOrderNotFound
	}
	return o, err
}

func visible(actor authctx.Principal, o *domain.Order) bool {
	return actor.IsAdmin() || o.UserID == actor.UserID
}

func (s *orderService) Create(ctx context.Context, actor authctx.Principal, planID string) (*domain.Order, error) {
	plan, err := s.plans.FindByID(ctx, planID)
	if errors.Is(err, apperror.ErrNotFound) || (err == nil && !plan.IsActive) {
		return nil, apperror.Validation(map[string]string{"plan_id": "Paket tidak tersedia"})
	} else if err != nil {
		return nil, err
	}

	now := s.now()
	var id string
	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		// Serialisasi pembuatan order: cek order tertunda & pemilihan nominal unik harus atomik.
		if err := s.locker.LockTx(ctx, "billing:order_create"); err != nil {
			return err
		}
		pending, err := s.orders.ListPendingByUser(ctx, actor.UserID)
		if err != nil {
			return err
		}
		for i := range pending {
			o := &pending[i]
			if !o.IsOverdue(now) {
				return domain.ErrOrderPending(o.ID)
			}
			// Lewat batas waktu tapi job lifecycle belum jalan → tandai expired sekarang.
			if err := o.Expire(now); err != nil {
				return err
			}
			if err := s.orders.Update(ctx, o); err != nil {
				return err
			}
		}

		taken, err := s.orders.TakenAmounts(ctx, domain.OrderAmount(plan.Price, domain.MinUniqueCode),
			domain.OrderAmount(plan.Price, domain.MaxUniqueCode), now)
		if err != nil {
			return err
		}
		code, err := domain.PickUniqueCode(plan.Price, taken, domain.MinUniqueCode+rand.IntN(domain.MaxUniqueCode))
		if err != nil {
			return err
		}
		o := domain.NewOrder(actor.UserID, plan, code, now)
		for range 8 {
			o.Code = domain.NewOrderCode(now, security.RandomCode(4))
			if err = s.orders.Create(ctx, o); !errors.Is(err, domain.ErrOrderCodeTaken) {
				break
			}
		}
		if err != nil {
			return err
		}
		id = o.ID
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.find(ctx, id, false)
}

func (s *orderService) Get(ctx context.Context, actor authctx.Principal, id string) (*domain.Order, error) {
	o, err := s.find(ctx, id, false)
	if err != nil {
		return nil, err
	}
	if !visible(actor, o) {
		return nil, errOrderNotFound
	}
	return o, nil
}

func (s *orderService) List(ctx context.Context, f domain.OrderFilter) ([]domain.Order, int, error) {
	return s.orders.List(ctx, f)
}

func (s *orderService) UploadProof(ctx context.Context, actor authctx.Principal, id string, data []byte) (*domain.Order, error) {
	o, err := s.find(ctx, id, false)
	if err != nil {
		return nil, err
	}
	if o.UserID != actor.UserID {
		return nil, errOrderNotFound
	}
	if err := o.CanUploadProof(s.now()); err != nil {
		return nil, err
	}
	ext, err := storage.Detect(data, storage.ImageRule)
	if err != nil {
		return nil, err
	}
	key, err := s.files.SavePrivate(ctx, "proofs", data, ext)
	if err != nil {
		return nil, err
	}
	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		o, err := s.find(ctx, id, true)
		if err != nil {
			return err
		}
		if err := o.AttachProof(key, s.now()); err != nil {
			return err
		}
		return s.orders.Update(ctx, o)
	})
	if err != nil {
		return nil, err
	}
	return s.find(ctx, id, false)
}

func (s *orderService) OpenProof(ctx context.Context, actor authctx.Principal, id string) (*ProofFile, error) {
	o, err := s.Get(ctx, actor, id)
	if err != nil {
		return nil, err
	}
	if o.ProofPath == "" {
		return nil, apperror.NotFound("Bukti pembayaran belum diunggah")
	}
	body, mod, err := s.files.OpenPrivate(ctx, o.ProofPath)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, apperror.NotFound("File bukti pembayaran tidak ditemukan")
	} else if err != nil {
		return nil, err
	}
	ext := strings.ToLower(path.Ext(o.ProofPath))
	return &ProofFile{Body: body, ModTime: mod, ContentType: contentTypeFor(ext), Name: "bukti-" + o.Code + ext}, nil
}

func contentTypeFor(ext string) string {
	for ct, e := range storage.ImageRule.Types {
		if e == ext {
			return ct
		}
	}
	if ext == ".jpeg" {
		return "image/jpeg"
	}
	return "application/octet-stream"
}

func (s *orderService) Approve(ctx context.Context, actor authctx.Principal, id string) (*domain.Order, error) {
	var approved *domain.Order
	var sub *domain.Subscription
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		o, err := s.find(ctx, id, true)
		if err != nil {
			return err
		}
		if err := o.CanApprove(); err != nil {
			return err
		}
		sub, err = s.subs.Grant(ctx, o.UserID, o.PlanID, 0)
		if err != nil {
			return err
		}
		if err := o.Approve(actor.UserID, sub.ID, s.now()); err != nil {
			return err
		}
		approved = o
		return s.orders.Update(ctx, o)
	})
	if err != nil {
		return nil, err
	}
	// Dicatat setelah commit: Recorder tidak boleh menggagalkan (atau meracuni) transaksi utama.
	s.audit.Record(ctx, actor.UserID, "order.approve", "order:"+id, map[string]any{
		"code": approved.Code, "amount": approved.Amount, "user_id": approved.UserID,
		"subscription_id": sub.ID, "ends_at": sub.EndsAt,
	})
	return s.find(ctx, id, false)
}

func (s *orderService) Reject(ctx context.Context, actor authctx.Principal, id, reason string) (*domain.Order, error) {
	reason, err := domain.NormalizeRejectReason(reason)
	if err != nil {
		return nil, err
	}
	var rejected *domain.Order
	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		o, err := s.find(ctx, id, true)
		if err != nil {
			return err
		}
		if err := o.Reject(actor.UserID, reason, s.now()); err != nil {
			return err
		}
		rejected = o
		return s.orders.Update(ctx, o)
	})
	if err != nil {
		return nil, err
	}
	s.audit.Record(ctx, actor.UserID, "order.reject", "order:"+id, map[string]any{
		"code": rejected.Code, "user_id": rejected.UserID, "reason": reason,
	})
	return s.find(ctx, id, false)
}
