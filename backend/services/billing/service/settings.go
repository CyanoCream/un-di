package service

import (
	"context"
	"errors"
	"strings"

	"undangan/kernel/apperror"
	auditport "undangan/kernel/audit"
	"undangan/kernel/authctx"
	"undangan/kernel/normalize"
	"undangan/services/billing/domain"
)

type SettingsService interface {
	// Payment = pengaturan QRIS; nilai kosong bila belum pernah diisi.
	Payment(ctx context.Context) (*domain.PaymentSettings, error)
	UpdatePayment(ctx context.Context, actor authctx.Principal, in domain.PaymentSettings) (*domain.PaymentSettings, error)
}

type settingsService struct {
	repo  domain.SettingsRepository
	audit auditport.Recorder
}

func NewSettingsService(repo domain.SettingsRepository, audit auditport.Recorder) SettingsService {
	return &settingsService{repo: repo, audit: audit}
}

func (s *settingsService) Payment(ctx context.Context) (*domain.PaymentSettings, error) {
	ps, err := s.repo.GetPayment(ctx)
	if errors.Is(err, apperror.ErrNotFound) {
		return &domain.PaymentSettings{}, nil
	}
	return ps, err
}

// NormalizePaymentSettings merapikan & memvalidasi input admin.
func NormalizePaymentSettings(in domain.PaymentSettings) (domain.PaymentSettings, error) {
	f := apperror.Fields{}
	out := domain.PaymentSettings{
		QRISImage:    strings.TrimSpace(in.QRISImage),
		MerchantName: strings.TrimSpace(in.MerchantName),
		Instructions: strings.TrimSpace(in.Instructions),
	}
	if len(out.QRISImage) > 500 {
		f.Add("qris_image", "URL gambar QRIS terlalu panjang")
	} else if out.QRISImage != "" && !strings.HasPrefix(out.QRISImage, "/") &&
		!strings.HasPrefix(out.QRISImage, "https://") && !strings.HasPrefix(out.QRISImage, "http://") {
		f.Add("qris_image", "URL gambar QRIS tidak valid")
	}
	if len([]rune(out.MerchantName)) > 100 {
		f.Add("merchant_name", "Nama merchant maksimal 100 karakter")
	}
	if len([]rune(out.Instructions)) > 2000 {
		f.Add("instructions", "Instruksi maksimal 2000 karakter")
	}
	if raw := strings.TrimSpace(in.AdminWhatsApp); raw != "" {
		out.AdminWhatsApp = normalize.Phone(raw)
		if !strings.HasPrefix(out.AdminWhatsApp, "628") {
			out.AdminWhatsApp = ""
			f.Add("admin_whatsapp", "Nomor WhatsApp tidak valid (contoh: 6281234567890)")
		}
	}
	return out, f.Err()
}

func (s *settingsService) UpdatePayment(ctx context.Context, actor authctx.Principal, in domain.PaymentSettings) (*domain.PaymentSettings, error) {
	ps, err := NormalizePaymentSettings(in)
	if err != nil {
		return nil, err
	}
	if err := s.repo.SavePayment(ctx, &ps); err != nil {
		return nil, err
	}
	s.audit.Record(ctx, actor.UserID, "settings.payment.update", "settings:"+domain.PaymentSettingsKey, map[string]any{
		"merchant_name": ps.MerchantName, "admin_whatsapp": ps.AdminWhatsApp, "qris_image": ps.QRISImage,
	})
	return &ps, nil
}
