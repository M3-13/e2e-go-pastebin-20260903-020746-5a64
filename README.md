# Pastebin-REST-API in Go

Eine kleine Pastebin-REST-API, ausschließlich mit der Go-Standardbibliothek
(`net/http`) umgesetzt. Pastes werden in einem thread-sicheren In-Memory-Store
gehalten, über eine zufällige, URL-sichere ID adressiert und können optional
ablaufen. Die API liefert saubere Statuscodes und JSON-Fehlerantworten der Form
`{"error": "..."}`.

## Tech-Stack

- **Sprache**: Go 1.23
- **Framework**: nur `net/http` (Standardbibliothek, keine externen Abhängigkeiten)
- **Tests**: `go test` mit `net/http/httptest`
- **Modul**: `pastebin` (siehe `go.mod`)

## Installation

Voraussetzung ist eine Go-Installation (>= 1.23). Es sind keine externen
Abhängigkeiten erforderlich:

```
go mod download
```

## Ausführen (Dev)

```
go run .
```

Der Server lauscht standardmäßig auf Port `8080`. Der Port kann über die
Umgebungsvariable `PORT` überschrieben werden:

```
$env:PORT=9090; go run .      # PowerShell
PORT=9090 go run .            # Unix
```

## Bauen (Produktion)

```
go build -o pastebin .
./pastebin
```

## Endpoints

Alle Antworten mit Body tragen `Content-Type: application/json`.

| Methode | Pfad            | Beschreibung                              |
|---------|-----------------|-------------------------------------------|
| POST    | `/pastes`       | Legt einen neuen Paste an                 |
| GET     | `/pastes/{id}`  | Holt einen einzelnen Paste inkl. `content`|
| GET     | `/pastes`       | Listet Metadaten aller Pastes auf         |
| DELETE  | `/pastes/{id}`  | Löscht einen Paste                        |
| GET     | `/healthz`      | Health-Check, antwortet `{"status":"ok"}` |

- `created_at`/`expires_at` werden als RFC3339 serialisiert; `expires_at` ist
  `null`, wenn kein Ablauf gesetzt wurde.
- Fehlerantworten haben immer die Form `{"error": "<generischer Text>"}` und
  enthalten niemals `content`, Stacktraces, Dateipfade oder interne Typnamen.

## Features

- Thread-sicherer In-Memory-Store (`internal/store`) mit `RWMutex`.
- Zufällige, URL-sichere Paste-IDs mit 128 Bit Entropie (16 Bytes aus
  `crypto/rand`, kodiert mit `base64.RawURLEncoding`).
- Optionale Ablaufzeit für Pastes.
- Begrenzung auf `1000` gleichzeitig gespeicherte Pastes (`ErrStoreFull`).
- Panic-Recovery-Middleware.
- Health-Check-Route `/healthz`.
