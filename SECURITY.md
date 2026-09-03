VERDICT: CHANGES_REQUESTED

## Sicherheitsprüfung — Pastebin-REST-API

Geprüfter Stand: die oben angegebenen Quelltexte. Ein externer Security-Scanner wurde nicht geliefert; es wurden keine Abhängigkeiten außerhalb der Go-Standardbibliothek sichtbar. Das Fehlen von Scanner-Ergebnissen ist kein Befund.

### 1. Mittel: Abgelaufene Pastes blockieren den Store und ermöglichen unauthentifizierten DoS

**Betroffene Stelle:** `internal/store/store.go`, Methode `Store.Create`

Der Store begrenzt die Anzahl der Pastes auf `MaxPastes` mit:

```go
if len(s.pastes) >= MaxPastes {
    return Paste{}, ErrStoreFull
}
```

Abgelaufene Pastes werden jedoch nur in `Get`, `List` und `Delete` entfernt — nicht in `Create`. Ein unauthentifizierter Angreifer kann 1000 Pastes mit sehr kurzer Ablaufzeit (z. B. `expires_in_seconds:1`) anlegen. Nach Ablauf bleiben sie in der Map, zählen aber weiter gegen `MaxPastes`. Jeder weitere `POST /pastes` liefert dauerhaft 503, bis zufällig jemand `GET /pastes` aufruft. Damit kann der Angreifer die Kernfunktion für alle anderen Nutzer blockieren.

Da keine Authentifizierung oder Rate-Limitierung existiert und nur 1000 kleine Requests nötig sind, ist das ein realer Verfügbarkeitsangriff.

**Fix-Konzept:**

Vor der Längenprüfung abgelaufene Einträge entfernen, z. B. durch eine interne purge-Funktion, die bereits von `List` genutzt wird:

```go
func (s *Store) purgeExpiredLocked(now time.Time) {
    for id, p := range s.pastes {
        if p.ExpiresAt != nil && now.After(*p.ExpiresAt) {
            delete(s.pastes, id)
        }
    }
}

func (s *Store) Create(...) (Paste, error) {
    s.mu.Lock()
    defer s.mu.Unlock()

    s.purgeExpiredLocked(time.Now())

    if len(s.pastes) >= MaxPastes {
        return Paste{}, ErrStoreFull
    }
    ...
}
```

Ergänzend sollte ein Test sicherstellen, dass nach Ablauf eines voll gefüllten Stores ein neuer Paste erstellt werden kann.

---

### 2. Mittel: HTTP-Server ohne Timeouts — anfällig für Slowloris/Verbindungs-DoS

**Betroffene Stelle:** `main.go`, `main()`

```go
if err := http.ListenAndServe(":"+port, newHandler()); err != nil {
    panic(err)
}
```

Der Server wird ohne `http.Server`-Konfiguration betrieben. Es fehlen `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout` und `IdleTimeout`. Ein Angreifer kann viele Verbindungen öffnen und Header sehr langsam senden, wodurch Goroutinen und Systemressourcen blockiert werden. Das Body-Limit von 1 MiB greift erst nach dem Header-Read und schützt nicht gegen langsame Header.

**Fix-Konzept:**

Einen expliziten `http.Server` mit realistischen Timeouts verwenden:

```go
srv := &http.Server{
    Addr:              ":" + port,
    Handler:           newHandler(),
    ReadHeaderTimeout: 5 * time.Second,
    ReadTimeout:       15 * time.Second,
    WriteTimeout:      15 * time.Second,
    IdleTimeout:       60 * time.Second,
}
if err := srv.ListenAndServe(); err != nil {
    panic(err)
}
```

Die konkreten Werte sollten zum erwarteten Deployment passen; wichtig ist, dass mindestens `ReadHeaderTimeout` gesetzt ist.

---

### 3. Niedrig: Overflows und fehlende Obergrenze bei `expires_in_seconds`

**Betroffene Stelle:** `internal/httpapi/create.go`, `CreateHandler`

```go
expiresIn = time.Duration(*req.ExpiresInSeconds) * time.Second
```

Auf 64-Bit-Systemen kann `*req.ExpiresInSeconds` nahe an `math.MaxInt64` liegen. Die Multiplikation mit `time.Second` (int64) läuft über und kann negative Werte ergeben, wodurch die Ablaufprüfung im Store:

```go
if expiresIn > 0 {
    exp := now.Add(expiresIn)
    p.ExpiresAt = &exp
}
```

einen eigentlich befristeten Paste ohne Ablauf speichert. Die Fehlerbehandlung für `expires_in_seconds <= 0` schützt nicht, weil der Überlauf erst nach der Validierung entsteht. Das ist primär ein Datenschutz-/Retention-Problem: Der Aufrufer kann sich nicht darauf verlassen, dass sehr große Werte wie erwartet befristet sind.

**Fix-Konzept:**

Eine sinnvolle Obergrenze validieren, bevor die Dauer berechnet wird. Beispiel:

```go
const maxExpirySeconds = int64(math.MaxInt64 / int64(time.Second))

if req.ExpiresInSeconds != nil && (*req.ExpiresInSeconds <= 0 || int64(*req.ExpiresInSeconds) > maxExpirySeconds) {
    writeError(w, http.StatusBadRequest, "expires_in_seconds must be greater than 0 and not exceed max duration")
    return
}
```

Je nach Produktanforderung kann auch eine deutlich kleinere maximale Ablaufzeit (z. B. ein Jahr) sinnvoll sein.

---

### 4. Niedrig: Kein verschlüsselter Transport im Produktcode

**Betroffene Stelle:** `main.go`, Start des HTTP-Servers

Der Dienst lauscht ausschließlich auf HTTP. Paste-Inhalte können sensitive Daten enthalten, und alle Anfragen/Antworten sind im Netzwerk im Klartext lesbar. Der Code selbst ist hier nicht exploitbar, aber das Deployment muss HTTPS erzwingen.

**Fix-Konzept:**

TLS-Terminierung über einen vorgelagerten Reverse-Proxy (z. B. nginx/Caddy) verwenden oder den Go-Server mit `ListenAndServeTLS` und korrekt konfigurierten Zertifikaten betreiben. Zusätzlich sollte dokumentiert sein, dass `PORT`-Deployments nur hinter TLS betrieben werden dürfen.

---

### Positive Befunde

- Keine hartkodierten Secrets, Tokens oder Passwörter sichtbar.
- Keine SQL-, Command-, Path- oder JSON-Injection; JSON wird typisiert dekodiert.
- Paste-IDs werden mit `crypto/rand` und 16 Byte (128 Bit Entropie) erzeugt und URL-sicher kodiert.
- Request-Body-Limit von 1 MiB wird vor vollständigem Einlesen erzwungen.
- Fehlerantworten sind generisch und enthalten keine internen Details, Stacktraces oder `content`.
- Es gibt keine Logging-Ausgabe von Paste-Inhalten.
- Panic-Recovery verhindert Prozessabsturz und liefert generisches 500-JSON.

### Gesamteinschätzung

Der Code erfüllt die geforderten Schutzziele weitgehend. Blockierende Schwachstellen wie Secrets, Injection, Auth-Bypass oder bekannte verwundbare Abhängigkeiten wurden nicht gefunden. Die beiden mittleren Findings betreffen die Verfügbarkeit und sollten vor einem Produktionseinsatz behoben werden. Daher: **CHANGES_REQUESTED**.