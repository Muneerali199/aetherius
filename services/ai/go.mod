module github.com/aetherius/platform/services/ai

go 1.22

require (
	github.com/go-chi/chi/v5 v5.1.0
	github.com/google/uuid v1.6.0
	github.com/rs/zerolog v1.33.0
)

require (
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	golang.org/x/sys v0.21.0 // indirect
)

replace github.com/aetherius/platform/pkg => ../../pkg
