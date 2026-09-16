// Composition root: baca config, siapkan infrastruktur, rakit semua modul (DDD), jalankan server.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"undangan/app/internal/adapters"
	"undangan/app/internal/config"
	"undangan/app/internal/server"
	"undangan/kernel/database"
	"undangan/kernel/notify"
	"undangan/kernel/security"
	"undangan/kernel/storage"
	"undangan/migrations"

	auditctl "undangan/services/audit/controller"
	auditrepo "undangan/services/audit/repository"
	auditsvc "undangan/services/audit/service"

	authctl "undangan/services/auth/controller"
	authjwt "undangan/services/auth/jwt"
	authrepo "undangan/services/auth/repository"
	authsvc "undangan/services/auth/service"

	userctl "undangan/services/user/controller"
	userrepo "undangan/services/user/repository"
	usersvc "undangan/services/user/service"

	billingctl "undangan/services/billing/controller"
	billingrepo "undangan/services/billing/repository"
	billingsvc "undangan/services/billing/service"

	"undangan/services/invitation/renderer"
	themectl "undangan/services/theme/controller"
	themerepo "undangan/services/theme/repository"
	themesvc "undangan/services/theme/service"

	invctl "undangan/services/invitation/controller"
	invdomain "undangan/services/invitation/domain"
	"undangan/services/invitation/importer"
	invrepo "undangan/services/invitation/repository"
	invsvc "undangan/services/invitation/service"
	"undangan/services/invitation/stationtoken"

	mediactl "undangan/services/media/controller"
	mediarepo "undangan/services/media/repository"
	mediasvc "undangan/services/media/service"

	dashctl "undangan/services/dashboard/controller"
	dashrepo "undangan/services/dashboard/repository"
	dashsvc "undangan/services/dashboard/service"
	landingctl "undangan/services/landing/controller"
	landingsvc "undangan/services/landing/service"
	notifyctl "undangan/services/notify/controller"
	notifysvc "undangan/services/notify/service"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(log)
	if err := run(log); err != nil {
		log.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if cfg.JWTSecret == "" {
		if !cfg.Dev {
			return errors.New("JWT_SECRET wajib diisi di production")
		}
		cfg.JWTSecret = security.RandomToken(48)
		log.Warn("JWT_SECRET kosong: memakai secret acak (semua sesi hilang saat restart)")
	}

	// ---- Infrastruktur ----
	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := database.Migrate(ctx, pool, migrations.FS, log); err != nil {
		return err
	}
	tx := database.NewTxManager(pool)
	files, err := storage.New(ctx, storage.Config{
		Driver: cfg.StorageDriver, LocalPublicDir: cfg.UploadDir, LocalPrivateDir: cfg.PrivateDir,
		S3: storage.S3Config{
			Endpoint: cfg.S3Endpoint, Region: cfg.S3Region, Bucket: cfg.S3Bucket, AccessKey: cfg.S3AccessKey,
			SecretKey: cfg.S3SecretKey, UseSSL: cfg.S3UseSSL, PathStyle: cfg.S3PathStyle, PublicBaseURL: cfg.S3PublicBaseURL,
		},
	})
	if err != nil {
		return err
	}
	log.Info("storage siap", "driver", cfg.StorageDriver)
	hasher := security.NewBcryptHasher()
	render := renderer.NewRenderer(cfg.ThemesDir, cfg.Dev)
	urlFor := cfg.InvitationURL

	// ---- Audit ----
	audit := auditsvc.New(auditrepo.NewPostgres(pool), log)

	// ---- Notifikasi (Telegram, opsional) ----
	users := userrepo.NewPostgres(pool)
	telegramOrders := &adapters.TelegramOrders{}
	notifyCfg := notifysvc.Config{
		BotToken: cfg.TelegramBotToken, ChatIDs: cfg.TelegramChatIDs, WebhookSecret: cfg.TelegramWebhookSecret,
		WebhookURL: cfg.TelegramWebhookURL, APIBaseURL: cfg.TelegramAPIBase, AdminURL: cfg.AdminURL, AppName: cfg.AppName,
	}
	var notifier notify.Notifier = notify.Nop{}
	var notifyService notifysvc.Service
	if notifyCfg.Enabled() {
		notifyService = notifysvc.New(notifyCfg, telegramOrders, adapters.SystemAdmin{Repo: users}, log)
		notifier = notifyService
	} else {
		log.Info("telegram nonaktif (TELEGRAM_BOT_TOKEN / TELEGRAM_CHAT_IDS kosong)")
	}

	// ---- Auth & User ----
	tokens, err := authjwt.NewHS256(cfg.JWTSecret, cfg.AccessTTL)
	if err != nil {
		return err
	}
	refreshRepo := authrepo.NewPostgres(pool)
	// SessionRevoker user → repository refresh token (hindari siklus konstruktor auth ↔ user).
	userService := usersvc.New(users, hasher, refreshRepo, audit, notifier, tx)
	accounts := adapters.Accounts{Repo: users, Svc: userService}
	authService := authsvc.New(tokens, refreshRepo, accounts, hasher, tx, cfg.RefreshTTL)

	// ---- Theme ----
	themeService := themesvc.New(themerepo.NewPostgres(pool), adapters.ThemeCatalog{Dir: cfg.ThemesDir}, audit, log)
	if err := themeService.Sync(ctx); err != nil {
		return err
	}

	// ---- Billing ----
	plans := billingrepo.NewPlans(pool)
	subs := billingrepo.NewSubscriptions(pool)
	orders := billingrepo.NewOrders(pool)
	locker := billingrepo.NewLocker(pool)
	quota := billingsvc.NewInvitationQuotaAdapter(subs, plans)

	// ---- Invitation ----
	invitations := invrepo.NewInvitations(pool)
	guests := invrepo.NewGuests(pool)
	invitationService := invsvc.NewInvitationService(invitations, quota, themeService, audit, tx, hasher)
	checkinLogs := invrepo.NewCheckinLogs(pool)
	guestService := invsvc.NewGuestService(invitationService, guests, invitations, importer.Parser{}, audit, tx, urlFor, checkinLogs)
	checkinService := invsvc.NewCheckinService(invitationService, invitations, guests, checkinLogs, stationtoken.New(cfg.JWTSecret, 12*time.Hour), hasher)
	giftService := invsvc.NewGiftService(invitationService, invitations, guests, invrepo.NewGifts(pool), files)
	wishService := invsvc.NewWishService(invitationService, invitations, guests, invrepo.NewWishes(pool))
	siteService := invsvc.NewSiteService(invitations, guests)

	planService := billingsvc.NewPlanService(plans, locker, audit, tx)
	subscriptionService := billingsvc.NewSubscriptionService(subs, plans, invitationService, locker, audit, tx)
	orderService := billingsvc.NewOrderService(orders, plans, subscriptionService, files, locker, audit, notifier, cfg.AdminURL, tx)
	telegramOrders.Svc = orderService
	settingsService := billingsvc.NewSettingsService(billingrepo.NewSettings(pool), audit)
	lifecycle := billingsvc.NewLifecycle(orders, subs, billingrepo.NewNotifications(pool), invitationService, locker, tx, log)
	userAdapter := billingsvc.NewUserAdapter(subscriptionService)

	if err := planService.EnsureDefaultPlans(ctx); err != nil {
		return err
	}
	if created, err := userService.EnsureSuperAdmin(ctx, cfg.SeedAdminName, cfg.SeedAdminEmail, cfg.SeedAdminPassword); err != nil {
		return err
	} else if created {
		log.Info("super admin dibuat", "email", cfg.SeedAdminEmail)
	}

	// ---- Media & Dashboard ----
	mediaController := mediactl.New(mediasvc.New(mediarepo.NewUploads(pool), mediarepo.NewMusic(pool), files, audit), files)

	// ---- Controllers ----
	authController := authctl.New(authService, accounts, !cfg.Dev)
	themeController := themectl.New(themeService)
	themeSite := invctl.NewThemeSiteController(themeService, render, cfg.Dev)
	domainCfg := invdomain.DomainConfig{BaseDomain: cfg.BaseDomain, CNAMETarget: cfg.CustomDomainCNAME, IPs: cfg.CustomDomainIPs}
	invitationController := invctl.NewInvitationController(invitationService, themeService, render, urlFor, domainCfg)
	customDomainService := invsvc.NewCustomDomainService(invitationService, invitations, quota, net.DefaultResolver, domainCfg, audit)
	siteController := invctl.NewSiteController(siteService, render, cfg.BaseDomain)

	registrars := []server.Registrar{}
	var csrfExempt []string
	if notifyService != nil {
		registrars = append(registrars, notifyctl.New(notifyService))
		csrfExempt = append(csrfExempt, notifyctl.WebhookPath) // dijaga secret token dari Telegram
	}
	deps := server.Deps{
		Log:            log,
		BaseDomain:     cfg.BaseDomain,
		AuthMiddleware: authController.Middleware,
		API: append([]server.Registrar{
			authController,
			userctl.New(userService, userAdapter, userAdapter, invitationController),
			auditctl.New(audit),
			themeController,
			invitationController,
			invctl.NewGuestController(guestService),
			invctl.NewWishController(wishService),
			invctl.NewCheckinController(checkinService),
			invctl.NewGiftController(giftService),
			invctl.NewCustomDomainController(customDomainService, invitationController),
			billingctl.New(planService, orderService, subscriptionService, settingsService),
			mediaController,
			dashctl.New(dashsvc.New(dashrepo.NewPostgres(pool))),
		}, registrars...),
		ThemePreview: themeSite.Preview,
		ThemeShared:  themeSite.ServeShared,
		ThemeAsset:   themeSite.ServeAsset,
		Uploads:      mediaController.ServeUploads,
		Site:         siteController,
		TLSAsk:       siteController.TLSAsk,
		CSRFExempt:   csrfExempt,
		AdminApp:     server.SPA(cfg.AdminDist, "Portal admin"),
		Landing: landingctl.New(landingsvc.New(adapters.LandingThemes{Svc: themeService}, adapters.LandingPlans{Svc: planService},
			adapters.LandingContact{Svc: settingsService},
			landingsvc.Config{AppName: cfg.AppName, PortalURL: cfg.PortalURL, BaseDomain: cfg.BaseDomain}), cfg.LandingDir, cfg.Dev),
		PortalApp: server.SPA(cfg.PortalDist, "Portal customer"),
	}

	go lifecycle.Start(ctx, time.Hour)
	if notifyService != nil {
		notifyService.Start(ctx)
		if err := notifyService.SetupWebhook(ctx); err != nil {
			log.Error("telegram setup webhook", "err", err)
		} else {
			log.Info("telegram siap", "mode", map[bool]string{true: "webhook", false: "polling"}[cfg.TelegramWebhookURL != ""], "chat_ids", len(cfg.TelegramChatIDs))
		}
	}

	public := &http.Server{Addr: cfg.Addr, Handler: server.NewPublic(deps), ReadHeaderTimeout: 5 * time.Second}
	internal := &http.Server{Addr: cfg.InternalAddr, Handler: server.NewInternal(deps), ReadHeaderTimeout: 5 * time.Second}

	errc := make(chan error, 2)
	for _, srv := range []*http.Server{public, internal} {
		go func() {
			log.Info("listening", "addr", srv.Addr, "base_domain", cfg.BaseDomain)
			if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errc <- err
			}
		}()
	}

	select {
	case <-ctx.Done():
	case err := <-errc:
		return err
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = public.Shutdown(shutdownCtx)
	_ = internal.Shutdown(shutdownCtx)
	return nil
}
