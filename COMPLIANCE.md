VERDICT: CHANGES_REQUESTED

## Prüfumfang

Bewertet wurde der sichtbare Stand der Go-Backend-Pastebin-API (`net/http`, In-Memory-Store, Tests). Reine Backend-API ohne öffentliche Web-UI, daher sind Impressums-, Cookie- und Barrierefreiheitspflichten nicht einschlägig. Relevant sind DSGVO und CRA. Nach dem sichtbaren Code und der Sprint-Spec sind die Kernfunktionen funktionsfähig; die gefundenen Punkte sind behebbar, daher kein `BLOCKED`.

---

## 1. DSGVO / Datenschutz

### 1.1 Rechtsgrundlage und Transparenz — **hoch**
Der Dienst verarbeitet nutzergenerierte Inhalte (`content`), die personenbezogene Daten enthalten können. Im sichtbaren Code und in der Sprint-Spec ist keine Rechtsgrundlage benannt und kein Datenschutzhinweis vorhanden.

**Maßnahme:**  
- Datei `README.md` oder neue Datei `PRIVACY.md` mit Datenschutzhinweis nach Art. 13 DSGVO ergänzen: Verantwortlicher, Zweck der Verarbeitung, Rechtsgrundlage (z. B. Art. 6 Abs. 1 lit. b DSGVO für den Upload durch den Nutzer selbst), Speicherdauer, Betroffenenrechte, Kontaktmöglichkeit.
- Zusätzlich Nutzungsbedingungen/AGB ergänzen, die den Uploader verpflichten, nur Daten mit erforderlicher Rechtsgrundlage hochzuladen (wichtig für Daten Dritter).

### 1.2 Speicherdauer und Löschkonzept — **mittel**
Pastes ohne `expires_in_seconds` bleiben bis zum Prozessende gespeichert. Eine Speicherdauer ist im sichtbaren Stand nicht dokumentiert. Da der Store nur In-Memory arbeitet, ist die Dauer faktisch durch den Prozess begrenzt, muss aber dokumentiert werden.

**Maßnahme:**  
- In `PRIVACY.md`/`README.md` aufnehmen: „Pastes werden ausschließlich flüchtig im Arbeitsspeicher gehalten und gehen bei einem Neustart verloren. Ein Ablauf kann über `expires_in_seconds` gewählt werden; ohne Ablauf bleibt der Paste bis zum Löschen per API oder bis zum Neustart gespeichert.“  
- Kein harter Standardablauf im Store einführen, damit das bestehende Verhalten (`expires_at: null` ohne Ablauf) und die Tests erhalten bleiben.

### 1.3 Recht auf Berichtigung — **niedrig**
Ein eigener Update-Endpunkt fehlt. Berichtigung ist derzeit nur durch `DELETE` plus erneutes `POST` möglich.

**Maßnahme:**  
- Entweder in der Dokumentation ausdrücklich beschreiben, dass Berichtigung durch Löschen und Neuanlegen erfolgt, oder später einen `PUT /pastes/{id}`-Endpunkt mit Authentifizierung ergänzen. Für den aktuellen Sprint genügt die Dokumentation.

### 1.4 Betroffenenrechte und Kontaktweg — **mittel**
Die API bietet `GET /pastes/{id}` (Auskunft zu einem konkreten Paste) und `DELETE /pastes/{id}` (Löschung). Es fehlt aber ein dokumentierter Kontaktweg für Betroffenenanfragen.

**Maßnahme:**  
- In `README.md`/`PRIVACY.md` einen Abschnitt „Betroffenenrechte“ mit Kontakt-E-Mail oder Kontaktformular des Verantwortlichen ergänzen.

### 1.5 Positiv festgestellt
- Keine Logging-Ausgabe im Produktcode, die `content` verarbeitet (AC-16 erfüllt).
- Fehlerantworten enthalten keine internen Details, Stacktraces oder Dateipfade.
- Fehlerantworten enthalten den übermittelten `content` nicht (AC-15 erfüllt).
- Listen- und Create-Antworten enthalten `content` nicht.
- Die Paste-ID nutzt 16 Bytes aus `crypto/rand` und `base64.RawURLEncoding`; 128 Bit Entropie sind gegeben.

---

## 2. EU Cyber Resilience Act (CRA)

### 2.1 Fehlende Transportverschlüsselung — **hoch**
`main.go` startet den Server mit `http.ListenAndServe(":"+port, newHandler())` ohne TLS. Paste-Inhalte, die personenbezogene Daten enthalten können, würden bei direkter Exposition im Klartext übertragen.

**Maßnahme:**  
- In `main.go` TLS entweder direkt aktivieren (`http.ListenAndServeTLS` mit konfigurierbaren Zertifikatspfaden) oder explizit dokumentieren, dass vor dem Dienst zwingend ein TLS-terminierender Reverse-Proxy betrieben werden muss.  
- Zusätzlich `http.Server` mit `ReadTimeout`, `ReadHeaderTimeout`, `WriteTimeout` und `IdleTimeout` verwenden.

### 2.2 Unauthentifiziertes Löschen bei öffentlich abrufbarer ID-Liste — **hoch**
`GET /pastes` liefert alle IDs öffentlich aus. `DELETE /pastes/{id}` verlangt keinerlei Inhabernachweis. Damit kann jede Person zunächst alle IDs abrufen und anschließend systematisch alle Pastes löschen. Das ist ein erhebliches Risiko für Integrität und Verfügbarkeit.

**Maßnahme:**  
- Beim Erstellen einen zufälligen Owner-Token (z. B. weitere 16–32 Bytes aus `crypto/rand`, URL-sicher kodiert) generieren und ausschließlich in der Create-Antwort an den Nutzer zurückgeben.  
- Den Owner-Token **nicht** in `Paste`, `pasteMeta` oder `List` aufnehmen.  
- `DELETE /pastes/{id}` nur mit diesem Token zulassen, z. B. per Header `X-Paste-Token`.  
- Tests in `delete_test.go` entsprechend anpassen. Die übrigen Kernfunktionen bleiben funktionsfähig.

### 2.3 SBOM und Sicherheitsdokumentation — **mittel**
Sichtbar ist nur `go.mod` ohne externe Abhängigkeiten. Für die CRA-Konformität fehlt eine dokumentierte SBOM und eine Beschreibung der Sicherheitsannahmen.

**Maßnahme:**  
- Datei `SECURITY.md` ergänzen oder `README.md` erweitern:  
  - SBOM: „Keine externen Abhängigkeiten; nur Go-Standardbibliothek; Modul `pastebin`; Go-Version aus `go.mod`.“  
  - Sicherheitsannahmen: In-Memory-Speicher, 128-Bit-IDs, Request-Body-Limit, Store-Limit, Panic-Recovery.  
  - Update-/Patch-Verfahren und Supportzeitraum beschreiben.

### 2.4 Ratenbegrenzung und Serverhärtung — **niedrig**
Keine Ratenbegrenzung oder Begrenzung paralleler Anfragen. Das Store- und Body-Limit besteht, aber wiederholte Anfragen können Ressourcen binden.

**Maßnahme:**  
- Leichte Middleware in `main.go` oder `internal/httpapi` ergänzen: Token-Bucket/Ratenlimit pro Client-IP, optional `X-RateLimit-*`-Header.  
- Das Limit so wählen, dass Tests und normaler API-Betrieb nicht beeinträchtigt werden.

### 2.5 Recover-Middleware nach bereits geschriebener Antwort — **niedrig**
`Recover` ruft nach einem Panic immer `writeError` auf. Wenn der darunterliegende Handler bereits Header geschrieben hat, führt dies zu einem fehlerhaften zweiten Header-Schreiben.

**Maßnahme:**  
- In `internal/httpapi/middleware.go` prüfen, ob bereits eine Antwort begonnen wurde, z. B. durch einen `responseWriter`-Wrapper mit `wroteHeader`-Flag. Falls bereits geschrieben wurde, Verbindung schließen und nicht erneut `writeError` aufrufen.  
- Einen Panic-Test ergänzen, der erst teilweise schreibt.

---

## 3. EU AI Act

Keine KI-Funktion vorhanden. Der AI Act ist nicht anwendbar.

---

## 4. Pflichttexte und Web-UI

Keine öffentliche Web-UI, keine Cookie-Verarbeitung, kein Verkaufsangebot. Impressums-, Cookie-Banner- und Widerrufsbelehrungspflichten sind nicht einschlägig. Die Datenschutz-Transparenzpflicht gegenüber API-Nutzern bleibt bestehen, siehe DSGVO-Abschnitt 1.1.

---

## 5. Barrierefreiheit

Nicht anwendbar, da keine öffentliche Web-UI.

---

## 6. Sonstige Konformitätshinweise

### 6.1 AC-08: JSON-Fehler für nicht erkannte Pfade/Methoden — **mittel**
`http.NewServeMux` liefert bei unbekannten Pfaden oder Methoden standardmäßig Text-Antworten (`404 page not found`, `405 Method Not Allowed`), nicht die geforderte JSON-Struktur `{"error":"..."}`. Das betrifft z. B. `PUT /pastes` oder `/pastes/`.

**Maßnahme:**  
- In `main.go` eine Fallback-Route einrichten, z. B. `mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { writeError(w, http.StatusNotFound, "not found") })`.  
- Dabei prüfen, dass die spezifischen Methoden-Routen weiterhin Vorrang haben; das funktioniert mit Go 1.22+ ServeMux-Routing.

---

## Reconciliation

Die vorgeschlagenen Maßnahmen sind so gewählt, dass sie die Kernfunktionen nicht brechen:

- **TLS** kann über einen Reverse Proxy erfolgen oder direkt konfigurierbar ergänzt werden; der Betrieb bleibt möglich.  
- **Owner-Token für DELETE** schützt nur den Löschvorgang; `POST`, `GET /pastes/{id}`, `GET /pastes` und das Löschen mit gültigem Token funktionieren weiter.  
- **Datenschutzdokumentation** und **SBOM** sind rein dokumentarisch.  
- **Fallback-404 als JSON** ersetzt nur den bisherigen Mux-Standard und erfüllt AC-08.  
- **Standardablauf für Pastes ohne Ablauf** wird bewusst nicht als harter Codeeingriff gefordert, damit `expires_at: null` und die bestehenden Tests erhalten bleiben.