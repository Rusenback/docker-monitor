# Docker Monitor

TUI-pohjainen Docker konttien hallinta ja seurantasovellus.

## MVP Ominaisuudet

- [ ] Container listaus
- [ ] Start/Stop/Restart toiminnot
- [ ] Real-time stats
- [ ] Container logs

## Asennus

```bash
go mod download
go build -o dockermon ./cmd/dockermon
```

## Käyttö

```bash
./dockermon
```

## Kehitys

```bash
# Aja sovellus
go run ./cmd/dockermon

# Testit
go test ./...


# Build
go build -o dockermon ./cmd/dockermon
```

## Teknologiat

- Go 1.21+
- Docker API
- Bubbletea (TUI)
- Lipgloss (Styling)
