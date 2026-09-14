package domain

import (
	"fmt"
	"strings"
)

// DefaultGraceDays dipakai bila paket tidak menyebut masa tenggang.
const DefaultGraceDays = 7

// LandingThemeLimit = jumlah kartu tema di halaman utama (sisanya di /tema).
const LandingThemeLimit = 8

// BuildThemes memetakan tema ke view model + daftar kategori yang terisi + tema hero.
func BuildThemes(themes []Theme) (views []ThemeView, cats []Category, hero *ThemeView) {
	counts := map[string]int{}
	for _, t := range themes {
		if strings.TrimSpace(t.Slug) == "" {
			continue
		}
		cat := strings.ToLower(strings.TrimSpace(t.Category))
		if !ValidCategory(cat) {
			cat = ""
		}
		preview := t.PreviewURL
		if preview == "" {
			preview = "/_preview/" + t.Slug
		}
		name := strings.TrimSpace(t.Name)
		if name == "" {
			name = t.Slug
		}
		colors := make([]string, 0, 4)
		for _, c := range t.Colors {
			if isHexColor(c) && len(colors) < 4 {
				colors = append(colors, c)
			}
		}
		v := ThemeView{
			Slug: t.Slug, Name: name, Description: strings.TrimSpace(t.Description),
			Category: cat, CategoryLabel: CategoryLabel(cat),
			Thumbnail: t.Thumbnail, PreviewURL: preview, Colors: colors, Initials: Initials(name),
		}
		if cat == "" {
			v.CategoryLabel = ""
		}
		views = append(views, v)
		if cat != "" {
			counts[cat]++
		}
	}
	for _, c := range categoryOrder {
		if n := counts[c.Key]; n > 0 {
			cats = append(cats, Category{Key: c.Key, Label: c.Label, Count: n})
		}
	}
	for i := range views {
		if views[i].Thumbnail != "" {
			h := views[i]
			hero = &h
			break
		}
	}
	return views, cats, hero
}

// BuildPlans memetakan paket ke kartu harga dan menandai paket dengan harga per tamu termurah
// (hanya bila ada ≥ 2 paket berbayar — label berdasarkan data, bukan klaim).
func BuildPlans(plans []Plan) []PlanView {
	out := make([]PlanView, 0, len(plans))
	best, bestRatio, paid := -1, 0.0, 0
	anyDomain, anyCheckin := false, false
	for i, p := range plans {
		anyDomain = anyDomain || p.AllowCustomDomain
		anyCheckin = anyCheckin || p.AllowCheckin
		if p.Price > 0 && p.MaxGuests > 0 {
			paid++
			if r := float64(p.Price) / float64(p.MaxGuests); best < 0 || r < bestRatio {
				best, bestRatio = i, r
			}
		}
	}
	for i, p := range plans {
		grace := p.GraceDays
		v := PlanView{
			ID: p.ID, Name: p.Name, Description: p.Description, Price: p.Price,
			PriceLabel: FormatRupiah(p.Price), Duration: FormatDuration(p.DurationDays),
		}
		if p.Price == 0 {
			v.PriceLabel = "Gratis"
		}
		inv := "1 undangan digital"
		if p.MaxInvitations > 1 {
			inv = fmt.Sprintf("%d undangan digital", p.MaxInvitations)
		}
		v.Items = []PlanItem{
			{Text: inv, Included: true},
			{Text: fmt.Sprintf("Hingga %s tamu dengan link personal", FormatNumber(p.MaxGuests)), Included: true},
			{Text: "Aktif " + FormatDuration(p.DurationDays) + ", bisa diperpanjang", Included: true},
			{Text: "Semua tema, musik, galeri & countdown", Included: true},
			{Text: "RSVP, ucapan & amplop digital", Included: true},
			{Text: "Import tamu Excel & kirim via WhatsApp", Included: true},
			{Text: "Subdomain gratis", Included: true},
		}
		// Fitur pembeda hanya ditampilkan bila ada paket yang memilikinya (tanpa baris coret yang tak bisa dibeli).
		if anyDomain {
			v.Items = append(v.Items, PlanItem{Text: "Pakai domain sendiri", Included: p.AllowCustomDomain})
		}
		if anyCheckin {
			v.Items = append(v.Items, PlanItem{Text: "Check-in QR di venue & laporan kehadiran", Included: p.AllowCheckin})
		}
		if grace > 0 {
			v.Items = append(v.Items, PlanItem{Text: fmt.Sprintf("Masa tenggang %d hari", grace), Included: true})
		}
		if i == best && paid >= 2 {
			v.Highlight = true
			v.Badge = "Paling hemat per tamu"
		}
		out = append(out, v)
	}
	return out
}

// BuildFacts merangkum angka nyata untuk copy.
func BuildFacts(themes []ThemeView, plans []Plan) Facts {
	f := Facts{ThemeCount: len(themes), PlanCount: len(plans)}
	allCheck, allDomain := len(plans) > 0, len(plans) > 0
	for i, p := range plans {
		if i == 0 || p.Price < f.MinPrice {
			f.MinPrice = p.Price
		}
		if p.MaxGuests > f.MaxGuests {
			f.MaxGuests = p.MaxGuests
		}
		if p.GraceDays > f.GraceDays {
			f.GraceDays = p.GraceDays
		}
		if p.AllowCheckin {
			f.CheckinPlans = append(f.CheckinPlans, p.Name)
		} else {
			allCheck = false
		}
		if p.AllowCustomDomain {
			f.CustomDomainPlans = append(f.CustomDomainPlans, p.Name)
		} else {
			allDomain = false
		}
	}
	if f.GraceDays == 0 {
		f.GraceDays = DefaultGraceDays
	}
	f.AllPlansCheckin, f.AllPlansDomain = allCheck, allDomain
	return f
}

// planNote: "Tersedia di Paket Premium" / "" bila semua paket punya fitur tsb.
func planNote(names []string, all bool) string {
	if all || len(names) == 0 {
		return ""
	}
	return "Tersedia di " + JoinID(names)
}

// BuildFeatures = daftar keunggulan; fitur yang bergantung paket menyesuaikan data paket.
func BuildFeatures(f Facts, baseDomain string) []Feature {
	host := exampleSub(baseDomain)
	themeTitle := "Tema original, bukan template pasaran"
	if f.ThemeCount >= 20 {
		themeTitle = "20+ tema original, bukan template pasaran"
	} else if f.ThemeCount > 1 {
		themeTitle = fmt.Sprintf("%d tema original, bukan template pasaran", f.ThemeCount)
	}
	out := []Feature{
		{Icon: "palette", Wide: true, Title: themeTitle,
			Text: "Adat Nusantara, Islami, floral, modern, hingga elegan. Setiap tema dirancang sendiri—palet, tipografi, dan animasi pembukanya berbeda satu sama lain, jadi undangan Anda tidak terlihat seperti milik orang lain."},
		{Icon: "link", Wide: true, Title: "Link pribadi untuk setiap tamu",
			Text: "Setiap tamu mendapat tautan sendiri, misalnya " + host + "/budi-santoso, yang menyapa namanya. Nama yang tidak terdaftar ditolak, dan tersedia mode khusus tamu terdaftar untuk acara yang lebih privat."},
	}
	if len(f.CheckinPlans) > 0 {
		out = append(out, Feature{Icon: "qr", Wide: true, Title: "Check-in di venue, kehadiran tercatat real-time",
			Text: "Tamu cukup menunjukkan QR atau passcode dari undangannya. Penerima tamu memindai lewat HP tanpa perlu akun, tidak ada tamu yang terhitung dua kali, dan laporan kehadiran bisa diekspor ke Excel.",
			Note: planNote(f.CheckinPlans, f.AllPlansCheckin)})
	}
	out = append(out,
		Feature{Icon: "receipt", Title: "Konfirmasi kado & bukti transfer",
			Text: "Tamu bisa mengirim foto bukti transfer atau kabar kado langsung dari undangan. Bukti disimpan privat—hanya Anda yang melihatnya."},
		Feature{Icon: "gift", Title: "Amplop digital & alamat kado",
			Text: "Cantumkan rekening bank atau e-wallet dengan tombol salin, lengkap dengan alamat pengiriman kado."},
		Feature{Icon: "chat", Title: "RSVP & ucapan doa",
			Text: "Tamu mengonfirmasi kehadiran beserta jumlah orang, lalu meninggalkan ucapan yang bisa Anda sembunyikan bila perlu."},
		Feature{Icon: "whatsapp", Title: "Kirim via WhatsApp sekali klik",
			Text: "Pesan undangan dengan nama dan link personal tiap tamu sudah tersusun. Tinggal tekan kirim."},
		Feature{Icon: "sheet", Title: "Import daftar tamu dari Excel/CSV",
			Text: "Sudah punya daftar di Excel? Unggah saja. Nomor HP dirapikan dan data ganda dilewati otomatis."},
		Feature{Icon: "music", Title: "Musik latar",
			Text: "Pilih lagu dari pustaka atau unggah sendiri. Musik mulai saat undangan dibuka dan berhenti saat tab ditinggalkan."},
		Feature{Icon: "calendar", Title: "Countdown, kalender & peta",
			Text: "Hitung mundur menuju hari-H, tombol simpan ke Google Calendar, dan petunjuk arah lewat Google Maps."},
	)
	domainText := "Alamat " + exampleSub(baseDomain) + " sudah termasuk tanpa biaya tambahan."
	domainNote := ""
	if len(f.CustomDomainPlans) > 0 {
		domainText += " Ingin lebih personal? Pakai domain milik sendiri, seperti rakadannadia.com."
		domainNote = planNote(f.CustomDomainPlans, f.AllPlansDomain)
	}
	out = append(out,
		Feature{Icon: "globe", Title: "Subdomain gratis & domain sendiri", Text: domainText, Note: domainNote},
		Feature{Icon: "bolt", Title: "Ringan & hemat kuota tamu",
			Text: "Tanpa aplikasi dan tanpa halaman berat: font seperlunya, satu berkas gaya, dan foto dimuat saat dibutuhkan—tetap cepat di sinyal pas-pasan."},
		Feature{Icon: "leaf", Title: "Hemat kertas & ongkos kirim",
			Text: "Undangan sampai ke ratusan tamu, termasuk yang jauh di luar kota, tanpa biaya cetak dan kurir."},
	)
	return out
}

// BuildSteps = alur "Cara Kerja".
func BuildSteps() []Step {
	return []Step{
		{Title: "Daftar & pilih paket", Text: "Buat akun dalam satu menit, lalu pilih paket sesuai jumlah tamu Anda."},
		{Title: "Bayar via QRIS", Text: "Scan dari m-banking atau e-wallet apa pun dan unggah bukti bayar. Paket aktif begitu admin mengonfirmasi."},
		{Title: "Pilih tema & isi data", Text: "Pilih satu tema favorit, lalu isi data mempelai, acara, dan galeri. Ingin terima beres? Admin bisa bantu isikan."},
		{Title: "Bagikan ke tamu", Text: "Import daftar tamu, kirim link personal lewat WhatsApp, lalu pantau RSVP, ucapan, dan kehadiran dari dashboard."},
	}
}

// BuildFAQ = pertanyaan umum; jawaban menyesuaikan data paket & kontak.
func BuildFAQ(f Facts, plans []Plan, baseDomain, merchant string, hasWhatsApp bool) []FAQItem {
	pay := "Pembayaran memakai QRIS"
	if merchant != "" {
		pay += " atas nama " + merchant
	}
	pay += ". Scan dari aplikasi m-banking atau e-wallet apa pun, bayar sesuai nominal yang tertera (ada kode unik kecil agar mudah dicocokkan), lalu unggah foto bukti bayar. Admin mengonfirmasi secara manual dan paket langsung aktif setelah disetujui."
	if hasWhatsApp {
		pay += " Bila ingin lebih cepat, Anda juga bisa mengabari admin lewat WhatsApp."
	}
	out := []FAQItem{{Q: "Bagaimana cara pembayarannya?", A: pay}}

	dur := "Masa aktif mengikuti paket yang dipilih."
	if len(plans) > 0 {
		parts := make([]string, 0, len(plans))
		for _, p := range plans {
			parts = append(parts, p.Name+" aktif "+FormatDuration(p.DurationDays))
		}
		dur = "Sesuai paket: " + JoinID(parts) + "."
	}
	dur += " Masa aktif bisa diperpanjang kapan saja; perpanjangan ditambahkan ke sisa masa aktif, dan seluruh data undangan tetap aman."
	out = append(out, FAQItem{Q: "Berapa lama undangan saya aktif?", A: dur})

	out = append(out, FAQItem{Q: "Apakah tema bisa diganti setelah dipilih?",
		A: "Tema dipilih sekali lalu dikunci, supaya tampilan undangan yang sudah dibagikan tetap konsisten. Karena itu, silakan coba demo setiap tema terlebih dahulu. Bila benar-benar perlu mengganti, hubungi admin—admin dapat membantu menggantikan tema."})

	sub := exampleSub(baseDomain)
	if len(f.CustomDomainPlans) > 0 {
		where := "di semua paket"
		if !f.AllPlansDomain {
			where = "di " + JoinID(f.CustomDomainPlans)
		}
		out = append(out, FAQItem{Q: "Bisakah memakai domain sendiri?",
			A: "Bisa, " + where + ". Setiap undangan sudah mendapat subdomain gratis seperti " + sub + ". Untuk domain milik sendiri (misalnya rakadannadia.com), cukup arahkan DNS domain Anda sesuai petunjuk di dashboard; sertifikat HTTPS dipasang otomatis."})
	} else {
		out = append(out, FAQItem{Q: "Apa alamat undangan saya?",
			A: "Setiap undangan mendapat subdomain gratis seperti " + sub + " yang bisa Anda pilih sendiri selama belum dipublikasikan."})
	}

	if len(f.CheckinPlans) > 0 {
		where := "Di semua paket"
		if !f.AllPlansCheckin {
			where = "Di " + JoinID(f.CheckinPlans)
		}
		out = append(out, FAQItem{Q: "Bagaimana cara kerja check-in tamu di venue?",
			A: where + ", setiap tamu dengan link personal mendapat tiket berisi QR dan passcode. Di lokasi acara, penerima tamu membuka halaman check-in di HP, memasukkan PIN, lalu memindai QR atau mengetik passcode. Kehadiran tercatat real-time, tamu yang sama tidak terhitung dua kali, dan laporannya bisa diekspor ke Excel."})
	}

	out = append(out, FAQItem{Q: "Apakah bukti transfer dari tamu aman?",
		A: "Aman. Foto bukti transfer atau kado disimpan privat, tidak bisa dibuka lewat tautan publik, dan hanya dapat dilihat oleh Anda sebagai pemilik undangan serta admin."})

	guests := "Batas tamu mengikuti paket."
	if len(plans) > 0 {
		parts := make([]string, 0, len(plans))
		for _, p := range plans {
			parts = append(parts, p.Name+" hingga "+FormatNumber(p.MaxGuests)+" tamu")
		}
		guests = "Tergantung paket: " + JoinID(parts) + "."
	}
	guests += " Satu tamu adalah satu penerima link personal dan bisa mewakili beberapa orang, misalnya “Bapak Joko & Keluarga”."
	out = append(out, FAQItem{Q: "Berapa batas jumlah tamu?", A: guests})

	out = append(out, FAQItem{Q: "Apakah tamu perlu menginstal aplikasi?",
		A: "Tidak perlu. Undangan dibuka langsung di browser HP apa pun hanya dengan mengetuk link. Halamannya ringan, jadi tetap cepat walau sinyal sedang kurang bersahabat."})

	out = append(out, FAQItem{Q: "Apa yang terjadi bila masa aktif habis?",
		A: fmt.Sprintf("Anda mendapat masa tenggang %d hari untuk memperpanjang. Setelah itu undangan dinonaktifkan, tetapi datanya tidak langsung hilang—undangan dipulihkan begitu paket diperpanjang. Demi perlindungan data pribadi, data tamu dan ucapan dari undangan yang tidak diperpanjang dihapus permanen setelah satu tahun.", f.GraceDays)})

	if hasWhatsApp {
		out = append(out, FAQItem{Q: "Saya tidak sempat mengisi sendiri. Bisa dibantu?",
			A: "Bisa. Hubungi admin lewat WhatsApp; admin dapat membuatkan akun, memilihkan tema, dan mengisikan data undangan untuk Anda."})
	}
	return out
}

func exampleSub(base string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		base = "localhost"
	}
	return "raka-nadia." + base
}

func isHexColor(s string) bool {
	if len(s) != 4 && len(s) != 7 {
		return false
	}
	if s[0] != '#' {
		return false
	}
	for _, r := range s[1:] {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f' || r >= 'A' && r <= 'F') {
			return false
		}
	}
	return true
}
