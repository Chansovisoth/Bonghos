module github.com/Chansovisoth/Bonghos

go 1.26.6

replace golang.org/x/crypto => ./third_party/crypto

replace golang.org/x/sys => ./third_party/sys

replace golang.org/x/image => ./third_party/image

replace golang.org/x/term => ./third_party/term

replace rsc.io/qr => ./third_party/qr

require (
	github.com/creack/pty v1.1.24
	github.com/go-webauthn/webauthn v0.10.2
	github.com/gorilla/websocket v1.5.0
	github.com/klauspost/compress v1.18.7
	github.com/mattn/go-sqlite3 v1.14.22
	golang.org/x/crypto v0.21.0
	golang.org/x/image v0.0.0-00010101000000-000000000000
	golang.org/x/term v0.21.0
)

require (
	github.com/fxamacker/cbor/v2 v2.6.0 // indirect
	github.com/go-webauthn/x v0.1.9 // indirect
	github.com/golang-jwt/jwt/v5 v5.2.2 // indirect
	github.com/google/go-tpm v0.9.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	golang.org/x/sys v0.21.0 // indirect
	rsc.io/qr v0.0.0-00010101000000-000000000000 // indirect
)
