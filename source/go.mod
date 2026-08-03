module github.com/Chansovisoth/Bonghos

go 1.22

replace golang.org/x/crypto => ./third_party/crypto

replace golang.org/x/sys => ./third_party/sys

replace golang.org/x/image => ./third_party/image

replace golang.org/x/term => ./third_party/term

replace rsc.io/qr => ./third_party/qr

require (
	github.com/creack/pty v1.1.24
	github.com/gorilla/websocket v1.5.0
	github.com/klauspost/compress v1.17.9
	github.com/mattn/go-sqlite3 v1.14.22
	golang.org/x/crypto v0.0.0-00010101000000-000000000000
	golang.org/x/image v0.0.0-00010101000000-000000000000
	golang.org/x/term v0.21.0
)

require (
	golang.org/x/sys v0.21.0 // indirect
	rsc.io/qr v0.0.0-00010101000000-000000000000 // indirect
)
