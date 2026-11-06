module github.com/local/9yin-go-server

go 1.24

require (
	github.com/DATA-DOG/go-sqlmock v1.5.2
	github.com/go-sql-driver/mysql v1.9.3
	golang.org/x/text v0.28.0
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/iceisfun/golua/v2 v2.0.6-0.20260901081125-36de3be5500e
)

replace github.com/iceisfun/golua/v2 => ./third_party/golua
