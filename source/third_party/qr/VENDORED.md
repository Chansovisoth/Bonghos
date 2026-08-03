# rsc.io/qr (vendored)

Upstream: https://github.com/rsc/qr — BSD-3-Clause (see LICENSE).

Vendored for the same reason as the other dependencies here: Bonghos builds
offline with `GOPROXY=direct`, and `rsc.io/qr` resolves through a vanity import
that requires fetching meta tags from rsc.io. A `replace` directive in
`source/go.mod` points at this copy.

Only the QR encoder is included. The upstream `qart` (image-art variants) and
`libqrencode` (C reference data) directories are omitted; they are unused and
carry image assets Bonghos has no need for.

Used by `internal/qrcode` to render the TOTP enrolment QR code.
