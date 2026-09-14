module undangan/app

go 1.26.0

require (
	undangan/kernel v0.0.0
	undangan/migrations v0.0.0
	undangan/services/audit v0.0.0
	undangan/services/auth v0.0.0
	undangan/services/billing v0.0.0
	undangan/services/dashboard v0.0.0
	undangan/services/invitation v0.0.0
	undangan/services/landing v0.0.0
	undangan/services/media v0.0.0
	undangan/services/theme v0.0.0
	undangan/services/user v0.0.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/golang-jwt/jwt/v5 v5.3.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.11.0 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/klauspost/compress v1.19.2 // indirect
	github.com/klauspost/cpuid/v2 v2.4.0 // indirect
	github.com/klauspost/crc32 v1.3.0 // indirect
	github.com/minio/crc64nvme v1.1.1 // indirect
	github.com/minio/md5-simd v1.1.2 // indirect
	github.com/minio/minio-go/v7 v7.3.0 // indirect
	github.com/philhofer/fwd v1.2.0 // indirect
	github.com/richardlehane/mscfb v1.0.7 // indirect
	github.com/richardlehane/msoleps v1.0.6 // indirect
	github.com/rs/xid v1.6.0 // indirect
	github.com/skip2/go-qrcode v0.0.0-20200617195104-da1b6568686e // indirect
	github.com/tiendc/go-deepcopy v1.7.2 // indirect
	github.com/tinylib/msgp v1.6.4 // indirect
	github.com/xuri/efp v0.0.1 // indirect
	github.com/xuri/excelize/v2 v2.11.0 // indirect
	github.com/xuri/nfp v0.0.2-0.20250530014748-2ddeb826f9a9 // indirect
	github.com/zeebo/xxh3 v1.1.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/crypto v0.57.0 // indirect
	golang.org/x/net v0.58.0 // indirect
	golang.org/x/sync v0.23.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/text v0.42.0 // indirect
	gopkg.in/ini.v1 v1.67.3 // indirect
)

replace (
	undangan/kernel => ../kernel
	undangan/migrations => ../migrations
	undangan/services/audit => ../services/audit
	undangan/services/auth => ../services/auth
	undangan/services/billing => ../services/billing
	undangan/services/dashboard => ../services/dashboard
	undangan/services/invitation => ../services/invitation
	undangan/services/landing => ../services/landing
	undangan/services/media => ../services/media
	undangan/services/theme => ../services/theme
	undangan/services/user => ../services/user
)
