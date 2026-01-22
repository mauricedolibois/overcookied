# Deep Dive: Overcookied Infrastructure Explained

Dieses Dokument erklärt jede einzelne Datei in deiner `infra` und `k8s` Struktur. Es ist so geschrieben, dass es nicht nur *was* passiert erklärt, sondern auch *warum* wir diese Ressourcen überhaupt brauchen.

---

## 🏗 Teil 1: Terraform (`infra/`)

Terraform ist wie ein Bauplan für dein "virtuelles Rechenzentrum" bei AWS. Wir definieren Ressourcen in Code (HCL), und Terraform baut sie.

### 📍 `infra/base/` (Das Fundament)
Dies ist die unterste Ebene. Sie muss zuerst existieren und ändert sich fast nie.

#### 1. `vpc.tf` (Virtual Private Cloud)
*   **Was ist das?** Stell dir das VPC als deine eigene, isolierte Insel im AWS-Internet vor. Niemand kommt rein, es sei denn, du baust Brücken (Gateways).
*   **Wichtige Ressourcen:**
    *   `aws_vpc`: Das Hauptnetzwerk.
    *   `aws_internet_gateway`: Die "Brücke" zum Internet, damit deine Server Updates laden können und erreichbar sind.
    *   `aws_subnet`: Unterteilungen deiner Insel. Du hast "Public Subnets" definiert, was bedeutet, dass Server hier direkt mit dem Internet kommunizieren können (günstiger als Private Subnets, da kein NAT Gateway nötig ist).
    *   `aws_route_table`: Die Verkehrsregeln. Sie sagen: "Alles, was ins Internet will, muss über das Gateway gehen."

#### 2. `ecr.tf` (Elastic Container Registry)
*   **Was ist das?** Das ist wie DockerHub, aber privat bei AWS. Ein Parkhaus für deine Docker-Images.
*   **Wichtige Ressourcen:**
    *   `aws_ecr_repository`: Wir erstellen zwei Repos: eines für das Backend und eines für das Frontend.
    *   `aws_ecr_lifecycle_policy`: Ein automatischer Müllschlucker. Er löscht alte Images (behält nur die letzten 10), damit du nicht für uralte Versionen bezahlst.

#### 3. `backend.tf` (Das Gedächtnis)
*   **Was ist das?** Hier weiß Terraform, wo es sich merken soll, was es schon gebaut hat.
*   **Details:**
    *   `backend "s3"`: Terraform speichert seinen Zustand ("State") in einer Datei auf AWS S3 (`infra/base/terraform.tfstate`). Das ist wichtig, damit Terraform beim nächsten Mal weiß: "Aha, das VPC mit der ID xyz habe ich schon gebaut, das muss ich nicht nochmal machen."
    *   `dynamodb_table`: Verhindert, dass zwei Leute gleichzeitig `terraform apply` ausführen und sich gegenseitig die Infrastruktur zerschießen ("Locking").

#### 4. `outputs.tf` (Das Schaufenster)
*   **Was ist das?** Terraform ist in Schichten geteilt. `outputs.tf` sagt, welche Informationen aus dieser Schicht (Base) für andere Schichten (EKS) sichtbar sein sollen.
*   **Inhalt:**
    *   Wir exportieren `vpc_id`, `public_subnet_ids` und die `ecr_backend_url`.
    *   Warum? Damit die nächste Schicht (`infra/eks`) weiß: "In welches Netzwerk soll ich das Cluster bauen?" und "Wo liegen die Docker Images?".

#### 5. `terraform.tfvars` & `variables.tf`
*   **Was ist das?** Deine Konfiguration. Statt "eu-central-1" überall hardzucoden, schreiben wir `var.aws_region`. Das macht den Code sauberer und wiederverwendbar.
*   **Wichtige Variablen im EKS-Layer:**
    *   `valkey_node_type`: ElastiCache Node-Größe (default: `cache.t3.micro` für Free Tier).
    *   `node_instance_types`: EC2 Instanztypen für EKS-Nodes.
    *   `dynamodb_table_users/games`: DynamoDB Tabellennamen.

---

### ☸️ `infra/eks/` (Das Kubernetes Cluster)
Hier wird es spannend. Dieser Code baut das eigentliche Cluster *in* das Fundament von oben.

#### 4. `eks.tf` (Elastic Kubernetes Service)
*   **Was ist das?** Die Definition deines Kubernetes-Clusters.
*   **Wichtige Ressourcen:**
    *   `aws_eks_cluster`: Die "Control Plane" (das Gehirn von K8s). AWS managt das für dich.
    *   `aws_eks_node_group`: Die "Worker Nodes" (die Muskeln). Das sind echte EC2-Instanzen (Server), auf denen deine Docker-Container laufen.
        *   *Scaling Config:* Wir erlauben 1-3 Server. Wenn viel los ist, startet AWS automatisch neue Server (Auto-Scaling).
    *   `aws_iam_role`: Digitale Ausweise. EKS braucht Rechte, um EC2-Server zu starten. Die Nodes brauchen Rechte, um Images aus dem ECR zu ziehen.

#### 5. `irsa.tf` (IAM Roles for Service Accounts)
*   **Was ist das?** Eine Brücke zwischen Kubernetes und AWS-Rechten.
*   **Warum?** Früher gab man dem *ganzen Server* Administrator-Rechte. Das ist gefährlich. Mit IRSA geben wir *nur dem einzelnen Pod* Rechte.
*   **Details:**
    *   `backend_pod_role`: Erlaubt NUR dem Backend-Pod, deine DynamoDB-Tabellen zu lesen/schreiben und Secrets zu holen. Wenn jemand das Frontend hackt, kommt er trotzdem nicht an die Datenbank.
    *   `aws_load_balancer_controller`: Erlaubt dem Load Balancer Controller (Software im Cluster), echte AWS Load Balancer zu erstellen.

#### 6. `security-groups.tf` (Firewalls)
*   **Was ist das?** Virtuelle Türsteher für deine Server.
*   **Regeln:**
    *   `cluster_sg`: Erlaubt Kommunikation zur Control Plane.
    *   `nodes_sg`: Erlaubt Kommunikation zwischen den Nodes und zu den Pods.
*   **Hinweis:** ElastiCache hat eine eigene Security Group (`valkey_sg` in `elasticache.tf`), die nur Traffic von EKS-Nodes auf Port 6379 erlaubt.

#### 8. `backend.tf` (Das Gedächtnis & Provider)
*   **Was ist das?** Macht zwei Dinge:
    1.  Speichert den State dieser Schicht in S3 (`infra/eks/terraform.tfstate`).
    2.  Konfiguriert die Werkzeuge (`providers`), die wir brauchen: `aws`, `kubernetes`, `helm`.
*   **Besonderheit:** Der `kubernetes` und `helm` Provider müssen sich erst beim neuen Cluster anmelden (authentifizieren), um Befehle auszuführen. Deswegen siehst du dort Code, der sich einen Token vom EKS-Cluster holt.

#### 9. `data.tf` (Der Spion)
*   **Was ist das?** Ermöglicht es dieser Terraform-Schicht, Informationen aus einer *anderen* Schicht zu lesen, ohne sie zu verändern.
*   **Details:**
    *   `data "terraform_remote_state" "base"`: Liest die `outputs.tf` aus der `infra/base`-Schicht.
    *   Dadurch weiß das `eks`-Skript: "Ah, das ist die VPC ID, die im Base-Layer erstellt wurde." So verbinden wir die beiden Schichten.

#### 10. `oidc.tf` (Der Türsteher-Ausweis)
*   **Was ist das?** Richtet "OpenID Connect" ein.
*   **Zweck:** Das ist die technische Grundlage für **IRSA** (siehe `irsa.tf`). Es erlaubt AWS (IAM), den Kubernetes-Pods zu vertrauen. Ohne diese Datei könnten Pods keine AWS-Rechte bekommen.

#### 11. `outputs.tf` (Nützliche Infos)
*   **Was ist das?** Gibt nach dem Bauen praktische Informationen aus.
*   **Inhalt:**
    *   `kubeconfig_command`: Der Befehl, den du kopieren kannst, um dich mit `kubectl` einzuloggen.
    *   `alb_controller_role_arn`: Die Rolle für den Load Balancer.
    *   `valkey_endpoint`: Die Adresse des ElastiCache Valkey Clusters für Redis-Verbindungen.
    *   `valkey_port`: Der Port (6379) für Valkey-Verbindungen.

#### 12. `elasticache.tf` (Distributed State)
*   **Was ist das?** AWS ElastiCache mit Valkey 8.0 als verteilter Cache für Matchmaking und Game State.
*   **Warum brauchen wir das?** Ohne ElastiCache würde jeder Backend-Pod seinen eigenen Matchmaking-Queue haben. Spieler auf Pod A würden nie mit Spielern auf Pod B gematcht werden.
*   **Wichtige Ressourcen:**
    *   `aws_elasticache_subnet_group`: Definiert, in welchen Subnets der Cache laufen soll.
    *   `aws_security_group`: Firewall-Regeln. Erlaubt nur Traffic von EKS-Nodes auf Port 6379.
    *   `aws_elasticache_replication_group`: Der eigentliche Valkey-Cluster.
        *   *Engine:* `valkey` (Redis-kompatibel, aber Open Source).
        *   *Version:* 8.0
        *   *Node Type:* `cache.t3.micro` (Free Tier eligible).
        *   *Single Node:* Keine Replicas für Kosteneinsparung.
*   **Verwendungszweck:**
    *   **Matchmaking Queue:** Redis Sorted Set für FIFO-Warteschlange.
    *   **Distributed Locking:** Verhindert Race Conditions beim Matching.
    *   **Game State:** JSON-Speicherung des Spielzustands.
    *   **Pub/Sub:** Event-Broadcasting zwischen Pods.

#### 13. `route53.tf` (Das Telefonbuch & Sicherheit)
*   **Was ist das?** Route 53 ist der DNS-Service von AWS. Er übersetzt `overcookied.de` in die IP-Adresse deines Load Balancers. Zudem wäre hier das Zertifikats-Management (HTTPS) geregelt.
*   **Aktueller Status:** In deinem Code ist fast alles **auskommentiert**.
*   **Warum?** Domains kosten Geld (auch bei AWS). Dein Free-Tier deckt das nicht ab.
*   **Was würde passieren, wenn man es aktiviert?**
    1.  `aws_route53_zone`: Erstellt eine "Hosted Zone". Das ist der Container für alle DNS-Einträge deiner Domain.
    2.  `aws_acm_certificate`: Bestellt ein kostenloses SSL/TLS Zertifikat für `*.overcookied.de`. Damit bekommst du das grüne Schloss im Browser (HTTPS).
    3.  `aws_route53_record` (Validation): Beweist AWS, dass dir die Domain wirklich gehört (DNS Challenge), damit sie das Zertifikat ausstellen.
    4.  `aws_route53_record` (Alias): Der wichtigste Eintrag. Er würde sagen: "Wer `app.overcookied.de` aufruft, geh bitte zu diesem AWS Load Balancer hier."

## ☸️ Teil 2: Kubernetes Manifeste (`k8s/`)

Wenn Terraform fertig ist, steht nur die leere Hülle (das Cluster). Die YAML-Dateien hier sagen dem Cluster, was es tun soll.

### 📂 Root `k8s/`

#### 8. `namespace.yaml`
*   **Was ist das?** Ein virtueller Arbeitsbereich namens `overcookied`. Es trennt deine App von System-Dingen.

#### 9. `ingress.yaml` (Der Türsteher)
*   **Was ist das?** Die Routing-Regeln für den AWS Load Balancer.
*   **Funktion:**
    *   Kommt eine Anfrage an `/api`? ➡️ Schick sie zum Backend.
    *   Kommt eine Anfrage an `/`? ➡️ Schick sie zum Frontend.
*   **Annotations:** Die Zeilen, die mit `alb.ingress.kubernetes.io` beginnen, sind Anweisungen an den Controller aus `helm.tf`. Sie konfigurieren den echten AWS Load Balancer (Health Checks, Internet-Zugriff, etc.).

---

### 📂 `k8s/backend/`

#### 10. `deployment.yaml`
*   **Was ist das?** Die Bauanleitung für deine Backend-Pods.
*   **Details:**
    *   `replicas: 2`: Wir wollen immer 2 Kopien haben (für Ausfallsicherheit).
    *   `serviceAccountName`: Verknüpfung zur IAM-Rolle aus `irsa.tf`.
    *   `env`: Umgebungsvariablen. Hier werden z.B. die DynamoDB-Tabellennamen übergeben.
        *   `REDIS_ENDPOINT`: Verbindung zum ElastiCache Valkey für verteiltes Matchmaking (aus `redis-config` ConfigMap).
        *   `JWT_SECRET`: Shared Secret für Token-Signierung über alle Replicas.
        *   `GOOGLE_REDIRECT_URL`: OAuth Callback URL (dynamisch nach ALB-Erstellung).

#### 11. `service.yaml`
*   **Was ist das?** Ein stabiler Netzwerk-Endpunkt.
*   **Warum?** Pods sterben und bekommen neue IPs. Der Service ist wie eine feste Telefonnummer, die Anrufe immer an die gerade lebenden Pods weiterleitet.

#### 12. `oauth-configmap.yaml` (Dynamische Config)
*   **Was ist das?** Eine Konfigurationsdatei, die wir zur Laufzeit ändern können.
*   **Trick:** Unser Deploy-Skript schreibt hier die echte URL des Load Balancers rein, *nachdem* er erstellt wurde. So weiß das Backend, wohin es nach dem Google Login redirecten muss.

#### 13. `redis-configmap.yaml` (ElastiCache Verbindung) ⭐️ *Neu*
*   **Was ist das?** ConfigMap für die Valkey/Redis-Verbindungsdaten.
*   **Inhalt:**
    *   `REDIS_ENDPOINT`: Die Adresse des ElastiCache Clusters (Format: `endpoint:6379`).
*   **Dynamische Aktualisierung:** Das Deploy-Skript (`deploy-app.ps1`) holt sich den Endpoint aus Terraform Output und aktualisiert diese ConfigMap automatisch.
*   **Fallback:** Wenn kein Valkey verfügbar ist, nutzt das Backend In-Memory Matchmaking (nicht skalierbar, aber funktional für Entwicklung).

---

### 📂 `k8s/frontend/`

#### 13. `deployment.yaml` & `service.yaml`
*   Ähnlich wie beim Backend, aber einfacher. Das Frontend braucht keine IAM-Rechte (Service Account), da es nur HTML/JS ausliefert und keine Datenbank direkt anfasst.

---

### 🚀 Zusammenfassung des Datenflusses
1.  **User** ruft `http://load-balancer-url/` auf.
2.  **AWS ALB** (konfiguriert durch `ingress.yaml`) sieht`/` und leitet an **Frontend Service** weiter.
3.  **Frontend Pod** liefert React-App aus.
4.  **React App** ruft `/api/games` auf.
5.  **AWS ALB** sieht `/api`, leitet an **Backend Service** weiter.
6.  **Backend Pod** (mit IAM Rolle aus `irsa.tf`) authentifiziert sich bei **DynamoDB** und holt Daten.
7.  Antwort geht den Weg zurück.

### 🎮 Matchmaking-Datenfluss (mit ElastiCache)
1.  **User** klickt "Find Match" im Frontend.
2.  **WebSocket** Nachricht `JOIN_QUEUE` geht an **Backend Pod A**.
3.  **Backend Pod A** fügt Spieler zur **ElastiCache Valkey** Queue hinzu (`ZADD`).
4.  **Backend Pod B** (oder A) führt Matchmaking aus:
    *   Holt Distributed Lock (`SETNX`).
    *   Prüft Queue auf 2+ Spieler.
    *   Erstellt Match und speichert Game State in Redis.
    *   Publiziert Match-Notification via **Pub/Sub**.
5.  Beide Backend Pods erhalten Notification und senden `GAME_START` an ihre lokalen Clients.
6.  **Game State** wird bei jedem Click atomar in Redis aktualisiert (`WATCH`/`MULTI`).
7.  Nach Spielende werden Stats in **DynamoDB** persistiert.
