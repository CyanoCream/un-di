# Katalog Tema — brief & aturan orisinalitas

## Aturan orisinalitas (wajib)
1. **Tidak boleh mirip referensi** (tamukita/invisimple, viding, our-wedding.link, canva, asmaradigital, aadigital):
   tidak memakai ilustrasi pasangan kartun, rumah joglo, bingkai gunungan/tetesan, bingkai kubah foto abu-abu,
   ornamen kepingan salju, dan tidak menyalin teks/judul section mereka (mis. "Nyuwun Pangestu lan Donga Restu", "Panggonan").
2. **Tidak boleh mirip tema lain di katalog.** Setiap tema wajib unik di **4 sumbu** sekaligus:
   - **Palet** (warna dominan berbeda),
   - **Pasangan font** (tidak dipakai tema lain — lihat daftar di bawah),
   - **Arketipe layout** (struktur visual section, bukan cuma warna),
   - **Konsep/ornamen & animasi pembuka**.
   Menggabungkan ide (palet dari A, layout dari B, konsep dari C) boleh selama hasil akhirnya tidak sama dengan satu tema pun.
3. Fungsi tetap lengkap lewat partial `_shared` (cover + nama tamu, salam, kutipan, mempelai, countdown, acara, live, cerita,
   galeri, hadiah + konfirmasi, RSVP & ucapan, tiket check-in, penutup, musik, dock). Urutan section boleh diubah bila konsep menuntut.
4. Jangan memakai kelas `.gate` di tema — dipakai halaman gerbang `_shared/gate.html`.
5. Performa: ≤ 2 keluarga Google Font (weight seperlunya), 1 CSS tema, ornamen SVG ringan, tanpa library JS tambahan.

## Kategori (`theme.json` → `category`)
`adat` · `islami` · `floral` · `modern` · `elegan` · `rustic` · `pastel` · `retro`

## Font yang sudah dipakai (jangan dipakai ulang)
| Tema | Font |
|---|---|
| jawa-sogan | Marcellus + Lora |
| islami-zamrud | Amiri + Mulish |
| floral-sage | Parisienne + EB Garamond |
| minimalis-monokrom | Bodoni Moda + DM Sans |
| royal-navy | Cinzel + Josefin Sans |
| bali-kamboja | Della Respira + Hanken Grotesk |
| minang-gadang | Rozha One + Work Sans |
| batak-gorga | Big Shoulders Display + Archivo |
| sunda-siger | Kurale + Nunito Sans |
| betawi-gigi-balang | Lilita One + Livvic |
| lavender-awan | Fraunces + Nunito |
| tropis-monstera | Gloock + Figtree |
| scrapbook-polaroid | Caveat + Courier Prime |
| surat-lilin | Pinyon Script + Newsreader |
| nur-pastel | Forum + Jost |
| midnight-burgundy | Instrument Serif + Manrope |
| champagne-marble | Italiana + Albert Sans |
| terracotta-boho | Young Serif + Outfit |
| groovy-70an | Shrikhand + Rubik |
| majalah-cinta | Abril Fatface + Libre Franklin |
Agen tema menambahkan barisnya sendiri di laporan; pilih pasangan yang benar-benar berbeda.

## Katalog

| # | Slug | Nama | Kategori | Palet | Arketipe layout | Konsep & pembuka |
|---|---|---|---|---|---|---|
| 1 | jawa-sogan | Jawa Sogan | adat | coklat sogan, emas, krem | kolom klasik berpanel | batik kawung/parang; **cover direvisi** (tanpa gunungan) |
| 2 | islami-zamrud | Islami Zamrud | islami | zamrud, emas, gading | panel melengkung mihrab | bintang geometris, lentera |
| 3 | floral-sage | Floral Sage | floral | sage, dusty pink, kertas | kartu lembut + medali oval | line-art bunga |
| 4 | minimalis-monokrom | Minimalis Monokrom | modern | off-white, hitam | editorial bernomor | tipografi raksasa |
| 5 | royal-navy | Royal Navy | elegan | navy, champagne | bingkai art-deco | kipas & sunburst |
| 6 | bali-kamboja | Bali Kamboja | adat | terakota, gading, hijau lumut | **gerbang terbelah** (candi bentar) lalu pita batu bertekstur | bunga kamboja + ukiran patra; pembuka: dua sisi gapura bergeser |
| 7 | minang-gadang | Minang Gadang | adat | marun, emas songket, hitam | **pita songket horizontal** berlapis | lengkung atap gonjong sebagai pembatas section; pembuka: tirai tenun terbuka |
| 8 | batak-gorga | Batak Gorga | adat | merah darah, hitam, putih tulang | **poster tipografi tebal** blok penuh | motif gorga tiga warna & pinggiran ulos; pembuka: kain ulos tersibak |
| 9 | sunda-siger | Sunda Siger | adat | mint, kuning gading, putih | **gulungan naskah** (panel memanjang bertekstur daluang) | mahkota siger & bambu; pembuka: siger naik bersinar |
| 10 | betawi-gigi-balang | Betawi Gigi Balang | adat | merah cabai, hijau pucuk, kuning | **grid kartu ceria** | pinggiran segitiga gigi balang, pola kembang kelapa; pembuka: pintu berlis gigi balang |
| 11 | midnight-burgundy | Midnight Burgundy | elegan | arang, burgundy, emas tua | **sinematik layar penuh** (section setinggi layar, judul ala film) | bunga gelap dramatis; pembuka: letterbox hitam membuka |
| 12 | champagne-marble | Champagne Marble | elegan | marmer putih, champagne, rose gold | **kartu akrilik bertumpuk** (panel kaca buram) | urat marmer CSS; pembuka: kartu kaca terangkat |
| 13 | terracotta-boho | Terracotta Boho | rustic | terakota, pasir, coklat susu | **kolase asimetris** + stiker | matahari bulat, pampas kering; pembuka: matahari terbit |
| 14 | groovy-70an | Groovy 70-an | retro | mustard, oranye bakar, coklat | **bergelombang** (pembatas wave, badge bulat) | vibe retro, piringan hitam; pembuka: vinyl berputar |
| 15 | majalah-cinta | Majalah Cinta | modern | merah editorial, hitam, putih | **sampul & spread majalah** (kolom, nomor edisi, pull-quote) | "The Wedding Issue"; pembuka: sampul terbalik |
| 16 | lavender-awan | Lavender Awan | pastel | lilac, biru bedak, krem | **kartu melayang di langit** | awan & bintang kecil; pembuka: awan tersibak |
| 17 | tropis-monstera | Tropis Monstera | floral | hijau tropis, koral, krem | **section diagonal terbelah** | daun monstera & palem line-art; pembuka: daun bergeser |
| 18 | scrapbook-polaroid | Scrapbook Polaroid | rustic | kertas kraft, biru denim, kuning selotip | **kolase polaroid miring** (fokus love story & galeri) | selotip, coretan tangan; pembuka: sampul album dibuka |
| 19 | surat-lilin | Surat Segel Lilin | elegan | vellum, biru debu, merah segel | **lembar surat bertumpuk** | amplop & segel lilin; pembuka: segel pecah & surat keluar |
| 20 | nur-pastel | Nur Pastel | islami | blush, krem, emas lembut | **kolom sempit + medali roset** | jendela roset 8 kelopak & bulan sabit garis; pembuka: bulan sabit terbit |
