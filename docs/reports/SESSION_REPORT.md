# 📋 Session Report: Overcookied Deployment & Troubleshooting

**Datum:** 13. Januar 2026  
**Projekt:** Overcookied - Idle Game auf AWS EKS

---

## 🎯 Zusammenfassung

In dieser Session haben wir das Overcookied-Projekt mit Custom Domain (overcookied.de) und HTTPS auf AWS EKS deployed und mehrere kritische Probleme gelöst.

---

## ✅ Durchgeführte Arbeiten

### 1. Custom Domain Setup (overcookied.de)
- Route 53 A-Record erstellt (Alias auf ALB)
- ACM SSL-Zertifikat validiert und an ALB angehängt
- Kubernetes Ingress mit HTTPS-Annotations konfiguriert

### 2. OAuth Authentication Fixes
- Cookie-basierte OAuth State-Verwaltung implementiert
- JWT Secret als shared Kubernetes Secret eingerichtet
- Ingress-Routing für Auth-Endpoints korrigiert

### 3. Frontend URL-Handling
- Relative URLs statt hardcodierter localhost-Adressen
- WebSocket-URL dynamisch zur Laufzeit ermittelt

### 4. Script-Verbesserungen
- `create-oauth-secret.ps1` um JWT-Secret-Erstellung erweitert
- `deploy-app.ps1` holt JWT aus AWS Secrets Manager
- `update-oauth-config.ps1` mit `-UpdateDNS` Flag

### 5. Dokumentation
- RUNBOOK.md mit vollständiger Option A (End-to-End Deployment)
- Troubleshooting-Sektion hinzugefügt
- Phase 5 für Custom Domain dokumentiert

---

## 🐛 Probleme & Lösungen

### Problem 1: Frontend zeigte `localhost:8080` in Production

**Symptom:** Nach dem Deployment versuchte das Frontend, API-Calls an `localhost:8080` zu senden.

**Ursache:** Next.js evaluiert `NEXT_PUBLIC_*` Umgebungsvariablen zur **Build-Zeit**, nicht zur Laufzeit. Die `getApiUrl()`-Funktion wurde während `npm run build` ausgewertet.

**Lösung:** 
- `getApiUrl()` Funktion entfernt
- Alle API-Calls verwenden jetzt relative URLs (`/auth/google/login`, `/api/leaderboard`)
- WebSocket-URL wird zur Laufzeit mit `window.location` berechnet

---

### Problem 2: OAuth `invalid_state` Error

**Symptom:** Nach Google-Login erschien "Invalid OAuth state" Fehler.

**Ursache:** OAuth State wurde in einer globalen Variable gespeichert. Bei mehreren Backend-Pods konnte der Callback an einen anderen Pod gehen, der den State nicht kannte.

**Lösung:**
- OAuth State als HTTP-Cookie speichern (`oauth_state`)
- Cookie mit `SameSite=Lax` und `Secure` Flag
- State aus Cookie lesen im Callback-Handler

---

### Problem 3: `/auth/callback` Routing-Konflikt

**Symptom:** OAuth Callback wurde an Backend statt Frontend geroutet.

**Ursache:** Ingress-Regel für `/auth/*` fing auch `/auth/callback` ab, das aber zum Frontend gehört.

**Lösung:**
- Spezifische Ingress-Pfade: `/auth/google`, `/auth/verify`, `/auth/logout` → Backend
- `/auth/callback` geht an Frontend (Default-Route)

---

### Problem 4: JWT 401 Unauthorized nach Login

**Symptom:** Nach erfolgreichem Login kam `401 Unauthorized` bei API-Calls.

**Ursache:** Jeder Backend-Pod generierte seinen eigenen JWT-Secret. Tokens von Pod A waren ungültig bei Pod B.

**Lösung:**
- Kubernetes Secret `jwt-secret` erstellt
- Alle Backend-Pods nutzen denselben Secret
- `deploy-app.ps1` erstellt Secret automatisch

---

### Problem 5: DNS-Auflösung funktioniert nicht lokal

**Symptom:** `curl https://overcookied.de` schlug fehl mit "Could not resolve host".

**Ursache:** Lokaler Router (Fritz.box) cached DNS negativ oder hat DNS-Rebinding-Protection.

**Diagnose:**
```powershell
# Via Google DNS funktioniert:
nslookup overcookied.de 8.8.8.8  # ✅ Gibt IPs zurück
curl https://overcookied.de/health --resolve overcookied.de:443:3.72.117.233  # ✅ HTTP 200
```

**Lösung:**
- Windows DNS-Cache leeren: `ipconfig /flushdns`
- Oder DNS-Server auf 8.8.8.8 ändern
- Dies ist ein lokales Netzwerkproblem, nicht AWS

---

## 📊 Finale Architektur

```
┌─────────────────────────────────────────────────────────────┐
│                      Internet                                │
│                         │                                    │
│                    overcookied.de                            │
│                         │                                    │
│              ┌──────────▼──────────┐                        │
│              │    Route 53 DNS     │                        │
│              │   (A-Record Alias)  │                        │
│              └──────────┬──────────┘                        │
│                         │                                    │
│              ┌──────────▼──────────┐                        │
│              │   ALB (HTTPS:443)   │                        │
│              │   ACM Certificate   │                        │
│              └──────────┬──────────┘                        │
│                         │                                    │
│    ┌────────────────────┼────────────────────┐              │
│    │                    │                    │              │
│    ▼                    ▼                    ▼              │
│ /api/*              /auth/*               /* (default)      │
│ /ws                 /health                                  │
│    │                    │                    │              │
│    └────────┬───────────┘                    │              │
│             ▼                                ▼              │
│     ┌───────────────┐              ┌─────────────────┐     │
│     │    Backend    │              │    Frontend     │     │
│     │   (Go:8080)   │              │  (Next.js:3000) │     │
│     │   2 Replicas  │              │   2 Replicas    │     │
│     └───────┬───────┘              └─────────────────┘     │
│             │                                               │
│     ┌───────▼───────┐                                      │
│     │   DynamoDB    │                                      │
│     │ CookieUsers   │                                      │
│     │ CookieGames   │                                      │
│     └───────────────┘                                      │
└─────────────────────────────────────────────────────────────┘
```

---

## 🔑 Wichtige Ressourcen

| Ressource | Wert |
|-----------|------|
| Domain | overcookied.de |
| EKS Cluster | overcookied-eks |
| Region | eu-central-1 |
| ACM Certificate ARN | `arn:aws:acm:eu-central-1:032073356456:certificate/75eb55b7-dde0-4aac-9836-278ed5d8063c` |
| Route 53 Hosted Zone | Z0075686230ZEJS28VZGB |

---

## 📝 Lessons Learned

1. **Next.js NEXT_PUBLIC_* sind Build-Zeit-Konstanten** - Niemals zur Laufzeit evaluieren
2. **Multi-Replica OAuth braucht shared State** - Cookies oder Redis statt globale Variablen
3. **JWT Secrets müssen geteilt werden** - Kubernetes Secrets für alle Pods
4. **Ingress-Routing ist pfadspezifisch** - Spezifische Pfade vor Wildcards definieren
5. **Lokale DNS-Probleme ≠ AWS-Probleme** - Immer mit öffentlichem DNS (8.8.8.8) verifizieren

---

## ✨ Status

**Die Anwendung ist vollständig deployed und funktionsfähig unter:**

🔗 **https://overcookied.de**
