# 🛠️ Lokale Entwicklungsumgebung

Diese Anleitung beschreibt, wie du Overcookied lokal entwickeln kannst, ohne die Produktionsumgebung oder AWS-Services zu beeinflussen.

---

## Schnellstart (mit Mocks)

Die einfachste Methode für lokale Entwicklung ist der **Mock-Modus**, der keine AWS-Services benötigt.

### 1. Backend starten

```bash
cd backend
cp .env.example .env
# Setze USE_MOCKS=true in der .env (Standard)
go run .
```

### 2. Frontend starten

```bash
cd frontend
npm install
npm run dev
```

### 3. Testen

Öffne `http://localhost:3000` im Browser.

---

## Mock-Modus

Der Mock-Modus ersetzt AWS-Services durch In-Memory-Implementierungen:

| Service | Mock-Verhalten |
|---------|----------------|
| **DynamoDB** | In-Memory-Speicher mit Sample-Daten (3 Benutzer, 3 Spiele) |
| **Redis/Valkey** | In-Memory-Queue für Matchmaking |
| **OAuth** | Funktioniert normal (benötigt Google Cloud Credentials) |

### Mock-Modus aktivieren

Setze in `backend/.env`:

```env
USE_MOCKS=true
```

### Mock-Modus deaktivieren (Produktion simulieren)

Setze in `backend/.env`:

```env
USE_MOCKS=false
AWS_ACCESS_KEY_ID=your-key
AWS_SECRET_ACCESS_KEY=your-secret
AWS_REGION=eu-central-1
REDIS_ENDPOINT=localhost:6379
```

---

## Voraussetzungen

- Go 1.24+
- Node.js 20+
- Google Cloud Console Zugang (für OAuth)

---

## 1. Google OAuth für localhost konfigurieren

### 1.1 Google Cloud Console öffnen

1. Gehe zu [Google Cloud Console - Credentials](https://console.cloud.google.com/apis/credentials)
2. Wähle das Projekt mit den bestehenden OAuth Credentials
3. Klicke auf die OAuth 2.0 Client ID (Web application)

### 1.2 Localhost URLs hinzufügen

**Autorisierte JavaScript-Quellen** - Füge hinzu:
```
http://localhost:3000
http://localhost:8080
```

**Autorisierte Weiterleitungs-URIs** - Füge hinzu:
```
http://localhost:8080/auth/google/callback
```

⚠️ **Wichtig:** Die Produktions-URLs (`https://overcookied.de`) NICHT entfernen!

---

## 2. Backend einrichten

### 2.1 Environment Variablen

```bash
cd backend
cp .env.example .env
```

Bearbeite `.env` und setze deine Google OAuth Credentials:

```env
# Google OAuth Credentials (aus Google Cloud Console)
GOOGLE_CLIENT_ID=123456789-abc.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=GOCSPX-xxxxx

# Redirect URL für OAuth Callback
GOOGLE_REDIRECT_URL=http://localhost:8080/auth/google/callback

# JWT Secret (beliebiger String)
JWT_SECRET=mein-lokales-jwt-secret-12345

# Frontend URL (kein trailing slash!)
FRONTEND_URL=http://localhost:3000

# Port
PORT=8080

# Mock Mode (true für lokale Entwicklung ohne AWS)
USE_MOCKS=true
```

### 2.2 Backend starten

```bash
cd backend
go run .
```

Das Backend startet auf `http://localhost:8080`

Du solltest folgende Logs sehen:
```
[MOCK] In-memory DynamoDB initialized for local development
[MOCK] Seeded 3 users and 3 games for local development
[MOCK] In-memory Redis/Valkey initialized for local development
[REDIS] Running in MOCK MODE - using in-memory matchmaking
```

---

## 3. Frontend einrichten

### 3.1 Dependencies installieren

```bash
cd frontend
npm install
```

### 3.2 Environment Variablen

Erstelle `frontend/.env.local`:

```env
NEXT_PUBLIC_API_URL=http://localhost:8080
```

### 3.3 Development Server starten

```bash
npm run dev
```

Das Frontend startet auf `http://localhost:3000`

---

## 4. Features testen

### 4.1 OAuth Login

1. Öffne `http://localhost:3000/login`
2. Klicke auf "Continue with Google"
3. Nach erfolgreichem Login wirst du zum Dashboard weitergeleitet

### 4.2 Matchmaking (Single-Player Test)

Mit Mock-Modus kannst du Matchmaking testen, indem du:
1. Zwei Browser-Fenster öffnest (oder normales + Inkognito)
2. In beiden einloggst (verschiedene Google-Accounts oder gleicher)
3. In beiden auf "Find Match" klickst

### 4.3 Leaderboard

Das Leaderboard zeigt die Mock-Benutzer:
- Alice Baker (1500 Punkte)
- Bob Chef (1200 Punkte)
- Charlie Cook (900 Punkte)

### 4.4 Game History

Die Game History zeigt Sample-Spiele für den Mock-Modus.

---

## 5. Troubleshooting

### OAuth Redirect funktioniert nicht

1. **Prüfe Google Cloud Console:** Sind `http://localhost:8080/auth/google/callback` und `http://localhost:3000` autorisiert?
2. **Prüfe .env:** Ist `GOOGLE_REDIRECT_URL=http://localhost:8080/auth/google/callback` gesetzt?
3. **Prüfe Frontend .env.local:** Ist `NEXT_PUBLIC_API_URL=http://localhost:8080` gesetzt?

### "redirect_uri_mismatch" Fehler

Die Redirect-URL in der `.env` muss **exakt** mit der in Google Cloud Console übereinstimmen.

### CORS-Fehler

Das Backend erlaubt automatisch Anfragen von `FRONTEND_URL`. Stelle sicher, dass:
- `FRONTEND_URL=http://localhost:3000` (ohne trailing slash!)
- Frontend tatsächlich auf Port 3000 läuft

### DynamoDB/Redis Fehler

Wenn du nicht im Mock-Modus bist (`USE_MOCKS=false`):
- AWS Credentials müssen korrekt konfiguriert sein
- Redis muss auf `localhost:6379` laufen (oder `REDIS_ENDPOINT` setzen)

---

## 6. Entwicklungs-Workflow

### Code-Änderungen

- **Backend:** Stoppe mit `Ctrl+C` und starte neu mit `go run .`
- **Frontend:** Hot-Reload ist aktiviert, Änderungen werden automatisch übernommen

### Mock-Daten anpassen

Bearbeite `backend/mocks/dynamo_mock.go` Funktion `seedData()` um andere Testdaten zu laden.

---

## 7. Nützliche Befehle

```bash
# Backend bauen
cd backend && go build .

# Backend Tests
cd backend && go test ./...

# Frontend Build (Produktion)
cd frontend && npm run build

# Frontend Linting
cd frontend && npm run lint
```
