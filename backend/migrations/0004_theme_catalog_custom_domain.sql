-- Katalog tema (kategori + thumbnail) dan domain sendiri untuk semua paket bawaan.

ALTER TABLE themes
    ADD COLUMN category  text NOT NULL DEFAULT '',
    ADD COLUMN thumbnail text NOT NULL DEFAULT '';

UPDATE plans SET allow_custom_domain = true WHERE name IN ('Paket Basic', 'Paket Premium');
