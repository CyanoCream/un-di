package domain

// Content = data undangan yang diisi admin/customer (JSONB invitations.content).
// Kontrak lengkap: docs/SPEC.md §2.
type Content struct {
	EventType   string     `json:"event_type"`
	Religion    string     `json:"religion"`
	CoupleOrder string     `json:"couple_order"`
	Cover       Cover      `json:"cover"`
	Opening     Opening    `json:"opening"`
	Quote       Quote      `json:"quote"`
	Groom       Person     `json:"groom"`
	Bride       Person     `json:"bride"`
	Events      []Event    `json:"events"`
	LoveStory   []Story    `json:"love_story"`
	Gallery     Gallery    `json:"gallery"`
	LiveStream  LiveStream `json:"live_stream"`
	Gift        Gift       `json:"gift"`
	RSVP        RSVP       `json:"rsvp"`
	Closing     Closing    `json:"closing"`
	Music       Music      `json:"music"`
	Share       Share      `json:"share"`
}

type Cover struct {
	Title      string `json:"title"`
	Photo      string `json:"photo"`
	Background string `json:"background"`
}

type Opening struct {
	Greeting string `json:"greeting"`
	Text     string `json:"text"`
}

type Quote struct {
	Text   string `json:"text"`
	Source string `json:"source"`
}

type Person struct {
	FullName       string `json:"full_name"`
	Nickname       string `json:"nickname"`
	Photo          string `json:"photo"`
	ChildOrder     string `json:"child_order"`
	Father         string `json:"father"`
	Mother         string `json:"mother"`
	FatherDeceased bool   `json:"father_deceased"`
	MotherDeceased bool   `json:"mother_deceased"`
	Instagram      string `json:"instagram"`
}

type Event struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Date      string `json:"date"`       // 2006-01-02
	StartTime string `json:"start_time"` // 15:04
	EndTime   string `json:"end_time"`   // "" = Selesai
	Timezone  string `json:"timezone"`   // WIB | WITA | WIT
	Venue     string `json:"venue"`
	Address   string `json:"address"`
	MapsURL   string `json:"maps_url"`
	Note      string `json:"note"`
}

type Story struct {
	Date  string `json:"date"`
	Title string `json:"title"`
	Text  string `json:"text"`
	Photo string `json:"photo"`
}

type Gallery struct {
	Photos   []string `json:"photos"`
	VideoURL string   `json:"video_url"`
}

type LiveStream struct {
	URL      string `json:"url"`
	Platform string `json:"platform"`
	Note     string `json:"note"`
}

type Gift struct {
	Enabled bool `json:"enabled"`
	// ConfirmationEnabled: tamu boleh mengirim bukti transfer / kado.
	ConfirmationEnabled bool          `json:"confirmation_enabled"`
	Text                string        `json:"text"`
	Accounts            []GiftAccount `json:"accounts"`
	Address             GiftAddress   `json:"address"`
}

type GiftAccount struct {
	Type     string `json:"type"` // bank | ewallet
	Provider string `json:"provider"`
	Number   string `json:"number"`
	Holder   string `json:"holder"`
}

type GiftAddress struct {
	Recipient string `json:"recipient"`
	Phone     string `json:"phone"`
	Address   string `json:"address"`
}

type RSVP struct {
	Enabled    bool `json:"enabled"`
	MaxPax     int  `json:"max_pax"`
	ShowWishes bool `json:"show_wishes"`
}

type Closing struct {
	Text    string   `json:"text"`
	SignOff string   `json:"sign_off"`
	From    string   `json:"from"`
	Family  []string `json:"family"`
}

type Music struct {
	Enabled bool   `json:"enabled"`
	URL     string `json:"url"`
	Title   string `json:"title"`
	StartAt int    `json:"start_at"`
}

type Share struct {
	WhatsAppTemplate string `json:"whatsapp_template"`
}

type religionDefaults struct {
	greeting, openingText, quote, quoteSource, signOff string
}

var defaultsByReligion = map[string]religionDefaults{
	"islam": {
		greeting:    "Assalamu'alaikum Warahmatullahi Wabarakatuh",
		openingText: "Maha Suci Allah yang telah menciptakan makhluk-Nya berpasang-pasangan. Dengan memohon rahmat dan ridho Allah SWT, kami bermaksud menyelenggarakan pernikahan putra-putri kami.",
		quote:       "Dan di antara tanda-tanda kekuasaan-Nya ialah Dia menciptakan untukmu pasangan hidup dari jenismu sendiri, supaya kamu merasa tenteram kepadanya, dan dijadikan-Nya di antaramu rasa kasih dan sayang.",
		quoteSource: "QS. Ar-Rum: 21",
		signOff:     "Wassalamu'alaikum Warahmatullahi Wabarakatuh",
	},
	"kristen": {
		greeting:    "Salam Sejahtera dalam Kasih Kristus",
		openingText: "Dengan penuh sukacita dan atas berkat Tuhan, kami mengundang Bapak/Ibu/Saudara/i untuk hadir dalam pernikahan kami.",
		quote:       "Demikianlah mereka bukan lagi dua, melainkan satu. Karena itu, apa yang telah dipersatukan Allah, tidak boleh diceraikan manusia.",
		quoteSource: "Matius 19:6",
		signOff:     "Tuhan Yesus memberkati",
	},
	"katolik": {
		greeting:    "Salam Damai Kristus",
		openingText: "Dengan penuh syukur atas rahmat Tuhan, kami mengundang Bapak/Ibu/Saudara/i untuk hadir dalam Sakramen Perkawinan kami.",
		quote:       "Kasih itu sabar; kasih itu murah hati; ia tidak cemburu. Ia menutupi segala sesuatu, percaya segala sesuatu, mengharapkan segala sesuatu, sabar menanggung segala sesuatu.",
		quoteSource: "1 Korintus 13:4-7",
		signOff:     "Berkah Dalem",
	},
	"hindu": {
		greeting:    "Om Swastyastu",
		openingText: "Atas Asung Kertha Wara Nugraha Ida Sang Hyang Widhi Wasa, kami bermaksud menyelenggarakan upacara pawiwahan putra-putri kami.",
		quote:       "Wahai pasangan suami-istri, semoga kalian tetap bersatu dan tidak pernah terpisahkan. Semoga kalian mencapai hidup penuh kebahagiaan.",
		quoteSource: "Rgveda X.85.42",
		signOff:     "Om Shanti Shanti Shanti Om",
	},
	"buddha": {
		greeting:    "Namo Buddhaya",
		openingText: "Dengan penuh sukacita, kami mengundang Bapak/Ibu/Saudara/i untuk hadir dan memberikan doa restu pada pernikahan kami.",
		quote:       "Suami dan istri yang saling setia, penuh kasih, dan menjalankan kebajikan akan hidup berbahagia bersama.",
		quoteSource: "Anguttara Nikaya",
		signOff:     "Sabbe Satta Bhavantu Sukhitatta",
	},
	"konghucu": {
		greeting:    "Wei De Dong Tian",
		openingText: "Dengan penuh sukacita, kami mengundang Bapak/Ibu/Saudara/i untuk hadir dan memberikan doa restu pada pernikahan kami.",
		quote:       "Hubungan suami istri yang harmonis ibarat alunan kecapi dan seruling.",
		quoteSource: "Shi Jing",
		signOff:     "Shan Zai",
	},
	"umum": {
		greeting:    "Dengan hormat,",
		openingText: "Tanpa mengurangi rasa hormat, kami mengundang Bapak/Ibu/Saudara/i untuk hadir dan memberikan doa restu pada hari bahagia kami.",
		quote:       "Cinta tidak saling memandang, tetapi bersama-sama memandang ke arah yang sama.",
		quoteSource: "Antoine de Saint-Exupéry",
		signOff:     "Terima kasih",
	},
}

// WithDefaults mengisi field kosong dengan nilai bawaan agar tema selalu tampil layak.
func (c Content) WithDefaults() Content {
	if c.EventType == "" {
		c.EventType = "pernikahan"
	}
	if c.CoupleOrder == "" {
		c.CoupleOrder = "groom_first"
	}
	d, ok := defaultsByReligion[c.Religion]
	if !ok {
		c.Religion = "islam"
		d = defaultsByReligion["islam"]
	}
	if c.Cover.Title == "" {
		if c.EventType == "ngunduh_mantu" {
			c.Cover.Title = "Ngunduh Mantu"
		} else {
			c.Cover.Title = "The Wedding of"
		}
	}
	if c.Opening.Greeting == "" {
		c.Opening.Greeting = d.greeting
	}
	if c.Opening.Text == "" {
		c.Opening.Text = d.openingText
	}
	if c.Quote.Text == "" {
		c.Quote.Text, c.Quote.Source = d.quote, d.quoteSource
	}
	if c.Closing.Text == "" {
		c.Closing.Text = "Merupakan suatu kebahagiaan dan kehormatan bagi kami apabila Bapak/Ibu/Saudara/i berkenan hadir dan memberikan doa restu."
	}
	if c.Closing.SignOff == "" {
		c.Closing.SignOff = d.signOff
	}
	if c.Closing.From == "" {
		c.Closing.From = "Kami yang berbahagia"
	}
	if c.RSVP.MaxPax <= 0 {
		c.RSVP.MaxPax = 2
	}
	if c.RSVP.MaxPax > 10 {
		c.RSVP.MaxPax = 10
	}
	if c.Gift.Text == "" {
		c.Gift.Text = "Doa restu Anda merupakan karunia yang sangat berarti bagi kami. Namun jika memberi adalah ungkapan tanda kasih, Anda dapat memberikannya melalui:"
	}
	if c.Share.WhatsAppTemplate == "" {
		c.Share.WhatsAppTemplate = DefaultWhatsAppTemplate
	}
	for i := range c.Events {
		if c.Events[i].Timezone == "" {
			c.Events[i].Timezone = "WIB"
		}
	}
	return c
}

const DefaultWhatsAppTemplate = "Kepada Yth.\nBapak/Ibu/Saudara/i\n*{nama}*\n\nTanpa mengurangi rasa hormat, perkenankan kami mengundang Bapak/Ibu/Saudara/i untuk menghadiri acara pernikahan kami.\n\nInfo lengkap acara:\n{link}\n\nMerupakan suatu kebahagiaan bagi kami apabila Bapak/Ibu/Saudara/i berkenan hadir dan memberikan doa restu.\n\nTerima kasih."
