module github.com/hirepilot/promptservice

go 1.24

toolchain go1.24.0

require github.com/hirepilot/shared v0.0.0

replace github.com/hirepilot/shared => ../shared

require (
	github.com/go-sql-driver/mysql v1.7.1 // indirect
	github.com/klauspost/compress v1.17.2 // indirect
	github.com/nats-io/nats.go v1.31.0 // indirect
	github.com/nats-io/nkeys v0.4.5 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	golang.org/x/crypto v0.6.0 // indirect
	golang.org/x/sys v0.18.0 // indirect
	golang.org/x/text v0.14.0 // indirect
)
