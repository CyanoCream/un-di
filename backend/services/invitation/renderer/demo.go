package renderer

import (
	"time"

	"undangan/services/invitation/domain"
)

// DemoContent dipakai katalog/preview tema. Tanggal acara selalu ~60 hari ke depan
// supaya countdown hidup.
func DemoContent() domain.Content {
	d := time.Now().AddDate(0, 0, 60).Format("2006-01-02")
	d2 := time.Now().AddDate(0, 0, 67).Format("2006-01-02")
	return domain.Content{
		EventType:   "pernikahan",
		Religion:    "islam",
		CoupleOrder: "groom_first",
		Groom: domain.Person{
			FullName: "Raka Aditya Pratama, S.T.", Nickname: "Raka", ChildOrder: "Putra pertama",
			Father: "Bapak Hendra Wijaya", Mother: "Ibu Sri Rahayu", Instagram: "raka.aditya",
		},
		Bride: domain.Person{
			FullName: "Nadia Putri Maharani, S.Pd.", Nickname: "Nadia", ChildOrder: "Putri kedua",
			Father: "Bapak Bambang Susilo", FatherDeceased: true, Mother: "Ibu Endang Lestari", Instagram: "nadiaputri",
		},
		Events: []domain.Event{
			{ID: "akad", Name: "Akad Nikah", Date: d, StartTime: "08:00", EndTime: "10:00", Timezone: "WIB",
				Venue: "Masjid Agung Baiturrahman", Address: "Jl. Pandanaran No. 126, Semarang"},
			{ID: "resepsi", Name: "Resepsi", Date: d, StartTime: "11:00", Timezone: "WIB",
				Venue: "Gedung Serbaguna Graha Santika", Address: "Jl. Pandanaran No. 116, Semarang"},
			{ID: "ngunduh", Name: "Ngunduh Mantu", Date: d2, StartTime: "10:00", EndTime: "14:00", Timezone: "WIB",
				Venue: "Kediaman Mempelai Pria", Address: "Jl. Kaliurang Km 7, Sleman, Yogyakarta"},
		},
		LoveStory: []domain.Story{
			{Date: "2019", Title: "Pertama Bertemu", Text: "Kami dipertemukan dalam sebuah kepanitiaan kampus. Obrolan singkat di sela rapat menjadi awal cerita kami."},
			{Date: "2023", Title: "Menjalin Komitmen", Text: "Setelah bertahun-tahun saling mengenal, kami memutuskan untuk melangkah bersama dengan lebih serius."},
			{Date: "2025", Title: "Lamaran", Text: "Dengan restu kedua keluarga, kami melangsungkan lamaran dan menentukan hari bahagia ini."},
		},
		LiveStream: domain.LiveStream{URL: "https://youtube.com/@contoh", Platform: "YouTube", Note: "Siaran langsung akad nikah"},
		Gift: domain.Gift{
			Enabled:             true,
			ConfirmationEnabled: true,
			Accounts: []domain.GiftAccount{
				{Type: "bank", Provider: "BCA", Number: "1234567890", Holder: "Raka Aditya Pratama"},
				{Type: "ewallet", Provider: "DANA", Number: "081234567890", Holder: "Nadia Putri Maharani"},
			},
			Address: domain.GiftAddress{Recipient: "Nadia Putri Maharani", Phone: "081234567890", Address: "Jl. Pemuda No. 45, Semarang, Jawa Tengah"},
		},
		RSVP:    domain.RSVP{Enabled: true, MaxPax: 4, ShowWishes: true},
		Closing: domain.Closing{From: "Kami yang berbahagia, Raka & Nadia beserta keluarga", Family: []string{"Keluarga Besar Bapak Hendra Wijaya", "Keluarga Besar Alm. Bapak Bambang Susilo"}},
		Music:   domain.Music{Enabled: true, Title: "Instrumental"},
	}
}
