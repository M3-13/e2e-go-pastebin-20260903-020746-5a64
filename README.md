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
Fehlerantworten haben immer die Form `{"error": "<generischer Text>"}` und
enthalten niemals `content`, Stacktraces, Dateipfade oder interne Typnamen.

### POST `/pastes`

Legt einen neuen Paste an.

**Request-Body:**

```json
{
  "content": "println(\"hello\")",
  "language": "go",
  "expires_in_seconds": 3600
}
```

- `content` (string, **erforderlich**) — darf nicht leer sein.
- `language` (string, optional) — z. B. `go`, `python`, `text`.
- `expires_in_seconds` (int, optional) — muss `> 0` sein, wenn angegeben.

**Antworten:**

| Status | Bedeutung                              |
|--------|----------------------------------------|
| 201    | Angelegt; Metadaten **ohne** `content` |
| 400    | Ungültiges JSON, leerer `content` oder `expires_in_seconds <= 0` |
| 413    | Request-Body größer als 1 MiB          |
| 503    | Store voll (mehr als 1000 Pastes)      |

```json
{
  "id": "3q2v7Nw...",
  "language": "go",
  "created_at": "2026-09-03T10:00:00Z",
  "expires_at": "2026-09-03T11:00:00Z"
}
```

### GET `/pastes/{id}`

Holt einen einzelnen Paste **inklusive** `content`.

| Status | Bedeutung                                  |
|--------|--------------------------------------------|
| 200    | Paste inkl. `content`                       |
| 404    | ID unbekannt **oder** Paste abgelaufen      |

```json
{
  "id": "3q2v7Nw...",
  "content": "println(\"hello\")",
  "language": "go",
  "created_at": "2026-09-03T10:00:00Z",
  "expires_at": "2026-09-03T11:00:00Z"
}
```

### GET `/pastes`

Listet die Metadaten aller nicht abgelaufenen Pastes auf. **Niemals** `content`.

```json
[
  {
    "id": "3q2v7Nw...",
    "language": "go",
    "created_at": "2026-09-03T10:00:00Z",
    "expires_at": "2026-09-03T11:00:00Z"
  }
]
```

### DELETE `/pastes/{id}`

Löscht einen Paste.

| Status | Bedeutung                                  |
|--------|--------------------------------------------|
| 204    | Gelöscht (kein Body)                       |
| 404    | ID unbekannt **oder** Paste abgelaufen      |

### GET `/healthz`

Health-Check, antwortet `{"status":"ok"}` mit Status 200.

**Hinweise zur Serialisierung:**

- `created_at`/`expires_at` werden als RFC3339 (`time.Time`) serialisiert.
- `expires_at` ist `null`, wenn kein Ablauf gesetzt wurde.

## Features

- Thread-sicherer In-Memory-Store (`internal/store`) mit `RWMutex`.
- Zufällige, URL-sichere Paste-IDs mit 128 Bit Entropie (16 Bytes aus
  `crypto/rand`, kodiert mit `base64.RawURLEncoding`).
- Optionale Ablaufzeit für Pastes.
- Begrenzung auf `1000` gleichzeitig gespeicherte Pastes (`ErrStoreFull`).
- Panic-Recovery-Middleware.
- Health-Check-Route `/healthz`.
