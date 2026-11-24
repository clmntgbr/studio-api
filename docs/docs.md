# Studio — documentation complète

Nov 23, 2026 · @Clement

## Vision et décisions

Studio transforme une vidéo (upload ou lien) en montage propre et en clips verticaux sous-titrés, avec une timeline que tu valides avant tout rendu. Usage perso, un seul utilisateur, produit complet dès le départ : ce document décrit la cible, et la section « Plan de construction » fixe l'ordre de codage.

Hors périmètre pour l'instant : multi-utilisateur, facturation, quotas, application mobile. L'architecture ne l'empêche pas, mais rien n'est construit pour.

| Sujet | Choix | Remarque |
| --- | --- | --- |
| API | Go 1.26, Fiber v3, GORM, Goose | Même socle qu'Uptime API : Clean Architecture, CQRS, outbox |
| Base | PostgreSQL | Timestamps média en millisecondes (BIGINT) |
| Broker | RabbitMQ, quorum queues | Remplace Kafka : jobs longs, un job à la fois par worker |
| Stockage | MinIO (compatible S3) | Aucune vidéo dans les messages, uniquement des clés |
| Temps réel | Centrifugo | Progression des étapes et mises à jour de l'éditeur |
| Transcription | AssemblyAI | Mots horodatés, diarisation, aucun GPU à gérer |
| LLM | Interface abstraite (OpenAI, DeepSeek, autre) | Sponsors, clips viraux, chapitres, titres |
| Rendu | FFmpeg + libass | Un seul encodage par sortie |
| Front | Next.js (React, App Router) | Éditeur de timeline, aperçu sans rendu |
| Auth | Clé API statique | Clerk ajoutable si le projet devient un produit |
| Workers | Go, sauf scenes et reframe en Python | Un conteneur `worker-xxx` par rôle |
| Entrées | Upload ou lien (yt-dlp) | Lien réservé aux vidéos dont tu as les droits |

## Fonctionnalités

Le produit couvre sept familles de fonctionnalités, toutes branchées sur la même timeline non destructive.

| Famille | Fonctionnalités |
| --- | --- |
| Ingestion | Upload multi-fichiers (multipart, reprenable), ajout par lien (yt-dlp), traitement en parallèle, proxy 720p pour l'aperçu, waveform |
| Détection | Silences (seuil auto ou manuel), mots de remplissage, répétitions de mots, faux départs, sponsors (LLM), pubs (PySceneDetect + noir + silence), chapitres |
| Timeline et validation | Timeline construite depuis les détections, chaque coupe avec sa raison, garder ou supprimer coupe par coupe ou par famille, coupes manuelles, versions, validation explicite avant rendu |
| Sous-titres | Animés mot à mot, mot prononcé en couleur, police, taille, contour, position, mots par ligne, majuscules, animation « pop », presets, aperçu exact dans le navigateur |
| Formats et recadrage | 16:9, 9:16, 1:1, 4:5, crop centré, gauche, droite, personnalisé, fond flouté, suivi de visage, bascule sur le locuteur actif |
| Clips | Moments viraux notés par LLM avec accroche et raison, bornes calées sur des fins de phrase, ajustement manuel, rendu par clip |
| Sorties | MP4 (master et clips), MP3, dossier de clips numérotés, FCPXML, EDL, OTIO, XML Premiere, SRT, ASS |

Quelques fonctions s'ajoutent autour du cœur :

- **Montage par le texte** : supprimer un mot ou une phrase dans le transcript crée une coupe manuelle.
- **Bandeau titre et incrustations** : texte dans un rectangle arrondi, logo, watermark, avec début et fin.
- **Qualité audio** : normalisation du volume (loudnorm), débruitage optionnel, fondus de 10 à 20 ms sur chaque coupe.
- **Textes générés** : titres, description et hashtags par LLM pour chaque clip.
- **Auto-zoom** léger sur les mots forts, désactivable.
- **Presets de sortie** : YouTube, Podcast, Vlog, Short vertical, un clic applique les réglages.

Le score de viralité reste une aide à la décision, pas une prédiction : il mesure des critères rédactionnels (accroche, autonomie, émotion, chute), pas les vues futures. Traduction des sous-titres et doublage sont prévus par l'architecture (un worker de plus) mais pas décrits ici.

## Principes d'architecture

La timeline est la seule représentation du montage ; tout le reste (sous-titres, rendus, exports, clips) en est dérivé. Seize principes en découlent.

1. **Sources immuables.** Le fichier original n'est jamais modifié ni ré-encodé en place.
2. **Audio extrait une seule fois** (Opus) ; silence et transcription réutilisent la même clé MinIO.
3. **Pas de découpage technique** de l'audio : silence et ASR travaillent sur le fichier complet.
4. **Détections brutes séparées des décisions.** Les workers produisent des détections ; seule la timeline décide ce qui est retiré.
5. **Une table de détections unique** (`detection`) pour silences, mots de remplissage, répétitions, faux départs, sponsors et pubs.
6. **Overrides ancrés sur la plage**, pas sur un index : un override « garder » survit à un recalcul parce qu'il se rattache par recouvrement de plage.
7. **Timeline versionnée.** Chaque reconstruction produit une version ; une version validée est figée.
8. **Validation avant rendu.** Aucun rendu coûteux sans timeline validée par l'utilisateur.
9. **SourceTime et OutputTime distincts.** Transcript, détections et clips vivent en SourceTime ; sous-titres et rendus en OutputTime.
10. **Clips définis en SourceTime.** Le rendu applique les coupes de la timeline à l'intérieur de la plage du clip.
11. **Un seul encodage** par sortie : crop, scale, ASS et overlays dans un seul graphe FFmpeg.
12. **Workers sans accès à la base.** Ils lisent et écrivent MinIO, reçoivent des jobs et renvoient des résultats par référence.
13. **Un seul orchestrateur.** Le core décide du job suivant ; les workers ne se connaissent pas.
14. **Idempotence partout** : clé de déduplication `(event_id, handler)`, empreinte de timeline, empreinte de rendu, marqueur MinIO pour le job AssemblyAI.
15. **Le calcul de la timeline est pur** (détections, config, overrides en entrée) : même entrée, même sortie, testable sans infrastructure.
16. **Coût en minutes** : le coût du système se mesure en minutes transcrites et minutes rendues.

Deux unités de temps coexistent :

```text
SOURCE TIME   00:00 -------------------------------- 30:00
                          coupes (timeline)
OUTPUT TIME   00:00 ---------------------- 24:32
```

Tous les temps média sont des entiers en millisecondes. Les conversions passent par une seule fonction du core, `SourceToOutput` et `OutputToSource`, appuyée sur les segments KEEP.

## Pipeline et workers

&#91;embedded content: pipeline · 12 workers, 1 validation\]

Quatre workers partent en parallèle après le média ; le core fusionne leurs résultats en timeline, et seule une timeline validée par l'utilisateur déclenche un rendu. Les workers ne s'appellent jamais entre eux : le core publie chaque job à la fin du précédent.

| Binaire | Langage | Queues | Base | Rôle |
| --- | --- | --- | --- | --- |
| `api` | Go | — | oui | HTTP, commandes et requêtes, URLs présignées |
| `worker` (core) | Go | `q.events`, `q.results`, `q.progress` | oui | Relais outbox, persistance des résultats, orchestration, timeline engine, plans de rendu, exports |
| `worker-fetch` | Go + yt-dlp | `q.fetch` | non | Téléchargement par lien |
| `worker-media` | Go + FFmpeg | `q.probe`, `q.audio`, `q.proxy`, `q.waveform` | non | Métadonnées, audio Opus, proxy 720p, waveform |
| `worker-silence` | Go + FFmpeg | `q.silence` | non | Plancher de bruit et silences bruts |
| `worker-transcribe` | Go | `q.transcribe` | non | Client AssemblyAI, mots horodatés, SRT |
| `worker-text` | Go | `q.text` | non | Remplissage, répétitions, faux départs |
| `worker-llm` | Go | `q.llm` | non | Sponsors, clips viraux, chapitres, titres |
| `worker-scenes` | Python | `q.scenes` | non | PySceneDetect, noir et silence : candidats de pubs |
| `worker-reframe` | Python | `q.reframe` | non | MediaPipe : trajectoire de crop |
| `worker-render` | Go + FFmpeg | `q.render` | non | Exécution du plan de rendu |

Les deux workers Python partagent les enveloppes et les fichiers JSON d'exemple des contrats avec les workers Go. Un nouveau worker (débruitage, traduction, doublage) s'ajoute avec sa queue et son type de job, sans modifier les autres.

## Messaging RabbitMQ

Trois exchanges séparent les jobs, les résultats et les événements de domaine ; chaque type de job a sa queue, avec retry et DLQ natifs.

| Exchange | Type | Routing key | Consommé par |
| --- | --- | --- | --- |
| `studio.jobs` | direct | `job.<type>` (ex. `job.silence`) | `q.<type>`, un worker par type |
| `studio.results` | topic | `result.<type>` | `q.results`, consommé par le core |
| `studio.events` | topic | `<aggregate>.<verb>` | `q.events`, handlers du core |
| `studio.progress` | fanout | aucune | `q.progress`, queue éphémère, relayée à Centrifugo |
| `studio.retry` | direct | `retry.<type>` | `q.<type>.retry` |

Les queues de jobs sont :

| Queue | Worker | Prefetch | Remarque |
| --- | --- | --- | --- |
| `q.fetch` | worker-fetch | 2 | Réseau, limité en débit |
| `q.probe`, `q.audio`, `q.waveform` | worker-media | 4 | Courts, CPU léger |
| `q.proxy` | worker-media | 1 | Encodage 720p, plus long |
| `q.silence` | worker-silence | 2 | Quelques secondes par heure d'audio |
| `q.transcribe` | worker-transcribe | 4 | Surtout de l'attente réseau |
| `q.text`, `q.llm` | worker-text, worker-llm | 4 | Texte seul |
| `q.scenes`, `q.reframe` | worker-scenes, worker-reframe | 1 | Lecture vidéo image par image |
| `q.render` | worker-render | 1 | Le plus lourd, un rendu par worker |

### Retry et DLQ

Chaque queue `q.x` est de type quorum, avec `q.x.retry` (TTL de délai) et `q.x.dlq`. En cas d'échec transitoire, le worker rejette le message sans le remettre ; il passe par `q.x.retry`, attend son TTL, puis revient dans `q.x`.

- Le nombre de passages se lit dans l'en-tête `x-death`. Au-delà de 5, le worker publie dans `q.x.dlq` et acquitte.
- Erreur non rejouable (fichier corrompu, paramètre invalide) : DLQ directe, sans retry.
- Trois paliers de délai (10 s, 1 min, 5 min) par queue de retry, sélectionnés selon le nombre d'essais.
- Un message en DLQ se rejoue depuis l'API (`retry` d'une étape), qui republie un job neuf avec une nouvelle `messageId`.

### Réglages à ne pas oublier

- **`consumer_timeout`** : 30 minutes par défaut, après quoi RabbitMQ ferme le canal du worker. Passer à 2 heures sur le broker ; à vérifier si la version permet un réglage par queue.
- **Publisher confirms** activés sur tous les publishers ; un job n'est considéré comme publié qu'après confirmation.
- **Acquittement manuel** uniquement après écriture MinIO et publication du résultat.
- **At-least-once** : tout handler est idempotent (voir contrats).
- **Outbox** : l'API et le core écrivent en base et dans la table `outbox_event` dans la même transaction ; le relais publie ensuite dans RabbitMQ, comme dans Uptime, seul le transport change.
- **Progression** : messages perdus acceptés, l'état réel vient toujours des résultats.

Variables : `RABBITMQ_URL`, `RABBITMQ_PREFETCH_<TYPE>`, `RETRY_MAX_ATTEMPTS`, `OUTBOX_POLL_INTERVAL`. Interface d'administration sur le port 15672 en local.

## Contrats de messages

Tous les messages partagent une enveloppe versionnée, et les résultats ne portent que des références MinIO, jamais de tableaux volumineux.

```json
{
  "messageId": "0b6f…",
  "type": "job.silence",
  "version": 1,
  "correlationId": "<mediaId>",
  "causationId": "<messageId du message déclencheur>",
  "occurredAt": "2026-11-23T14:00:00Z",
  "attempt": 1,
  "payload": { }
}
```

Règles de versionnage : un champ ajouté reste en `v1` ; un champ retiré ou changé de sens crée `v2`, et le core consomme les deux le temps de la migration. Les noms JSON sont en camelCase, les durées en millisecondes.

### Jobs et résultats

| Job | Entrée (payload) | Résultat (par référence) |
| --- | --- | --- |
| `fetch` | mediaId, url, maxSizeBytes, maxDurationSec | sourceKey, title, sizeBytes |
| `probe` | mediaId, sourceKey | durée, largeur, hauteur, fps, codecs, présence d'audio, rotation |
| `audio` | mediaId, sourceKey | audioKey (Opus mono) |
| `proxy` | mediaId, sourceKey | proxyKey (720p H.264, keyframe toutes les 0,5 s) |
| `waveform` | mediaId, audioKey | peaksKey (100 points par seconde) |
| `silence` | mediaId, audioKey, config de seuil | detectionsKey, noiseFloorDb, calculatedThresholdDb |
| `transcribe` | mediaId, audioKey, langue ou auto, diarisation, disfluencies | wordsKey, srtKey, language, providerJobId |
| `text` | mediaId, wordsKey, langue, config d'agressivité | detectionsKey (filler, repetition, false\_start) |
| `llm` | mediaId, wordsKey, tâches (sponsors, viral, chapitres, titres) | artifactsKey |
| `scenes` | mediaId, proxyKey | scenesKey, adCandidatesKey |
| `reframe` | mediaId, proxyKey | cropPathKey (suivi de visage par plage de temps) |
| `render` | renderId, plan de rendu complet | outputKey, durationMs, sizeBytes |

Le plan de rendu embarque tout ce dont le worker a besoin (segments KEEP, clés MinIO, ASS, overlays, crop, paramètres d'encodage). Le worker n'interroge jamais la base.

### Événements de domaine

Les événements passent par l'outbox puis `studio.events`. Les handlers du core déclenchent la suite et publient le temps réel.

| Événement | Déclenche |
| --- | --- |
| `media.source_ready` (upload: complete multipart et/ou webhook MinIO `object-created` ; fetch: `result.fetch.succeeded` — les deux appellent `MarkSourceReady`) | handler outbox → job `probe` |
| `media.probed` | jobs `audio` **et** `proxy` **en parallèle** (audio omis si pas de piste) |
| `media.audio_ready` | jobs `waveform` **et** `silence` **en parallèle** (+ `transcribe` dès que le worker studio existe) |
| `media.proxy_ready` | jobs `scenes` et `reframe` **en parallèle** (quand les workers studio existent) |
| `transcript.completed` | jobs `text` et `llm` |
| `detections.updated` | `timeline.rebuild_requested` (délai de 2 s pour regrouper) |
| `timeline.rebuilt` | notification temps réel, statut « à valider » |
| `timeline.validated` | autorise la création de rendus |
| `render.requested` | job `render` |
| `render.completed`, `render.failed` | notification, mise à jour du statut |

### Idempotence

- Chaque handler enregistre `(event_id, handler_name)` dans `processed_event` avant ses effets, comme dans Uptime.
- Le job `transcribe` est payant : avant de soumettre à AssemblyAI, le worker cherche `media/<id>/transcript/provider.job` dans MinIO. S'il existe, il reprend l'attente sur ce job au lieu d'en créer un autre.
- Une reconstruction de timeline dont l'empreinte (détections, config, overrides) n'a pas changé ne crée pas de version.
- Un rendu avec la même empreinte (timeline, preset, style, overlays) est réutilisé tel quel.

## Modèle de données

PostgreSQL porte l'état du produit ; les tables d'outbox et de déduplication (`outbox_event`, `processed_event`) sont reprises à l'identique d'Uptime API. Les migrations sont des fichiers Goose, source de vérité du schéma.

Trois choix par rapport à l'ancien document : une table `detection` unique, des `user_override` rattachés par plage, et un `timeline_decision` qui explique chaque coupe.

```sql
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TYPE project_status  AS ENUM ('draft','processing','ready','rendering','completed','failed');
CREATE TYPE media_status    AS ENUM ('pending','fetching','processing','ready','failed');
CREATE TYPE step_kind       AS ENUM ('fetch','probe','audio','proxy','waveform','silence','transcribe','text','llm','scenes','reframe');
CREATE TYPE step_status     AS ENUM ('pending','queued','running','completed','failed','skipped','cancelled');
CREATE TYPE detection_kind  AS ENUM ('silence','filler','repetition','false_start','sponsor','ad');
CREATE TYPE decision_action AS ENUM ('remove','suggest','keep');
CREATE TYPE decision_origin AS ENUM ('auto','user');
CREATE TYPE cut_reason      AS ENUM ('silence','filler','repetition','false_start','sponsor','ad','manual','micro');
CREATE TYPE timeline_status AS ENUM ('draft','validated','superseded');
CREATE TYPE render_kind     AS ENUM ('master','clip','audio','custom');
CREATE TYPE render_strategy AS ENUM ('full_render','stream_copy');
CREATE TYPE aspect_ratio    AS ENUM ('16:9','9:16','1:1','4:5','custom');
CREATE TYPE crop_mode       AS ENUM ('none','center','left','right','custom','fit_blur','face_track','speaker_track');

CREATE TABLE project (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name VARCHAR(255) NOT NULL,
  status project_status NOT NULL DEFAULT 'draft',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE media_file (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL REFERENCES project(id) ON DELETE CASCADE,
  position INTEGER NOT NULL DEFAULT 0,            -- ordre dans la timeline du projet
  source_kind VARCHAR(10) NOT NULL DEFAULT 'upload',  -- upload | url
  source_url TEXT,
  title TEXT,
  original_filename TEXT,
  storage_key TEXT,
  mime_type VARCHAR(100),
  duration_ms BIGINT,
  width INTEGER, height INTEGER, fps NUMERIC(8,3),
  video_codec VARCHAR(50), audio_codec VARCHAR(50),
  has_audio BOOLEAN,
  size_bytes BIGINT,
  status media_status NOT NULL DEFAULT 'pending',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (duration_ms IS NULL OR duration_ms >= 0)
);
CREATE INDEX idx_media_file_project ON media_file(project_id, position);

-- audio Opus, proxy, waveform, SRT, scenes, crop path : un asset par fichier dérivé
CREATE TABLE media_asset (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  media_file_id UUID NOT NULL REFERENCES media_file(id) ON DELETE CASCADE,
  kind VARCHAR(30) NOT NULL,    -- audio | proxy | peaks | words | srt | scenes | crop_path
  storage_key TEXT NOT NULL,
  size_bytes BIGINT,
  meta JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (media_file_id, kind)
);

CREATE TABLE media_configuration (
  media_file_id UUID PRIMARY KEY REFERENCES media_file(id) ON DELETE CASCADE,
  silence_enabled BOOLEAN NOT NULL DEFAULT TRUE,
  threshold_mode VARCHAR(10) NOT NULL DEFAULT 'auto',        -- auto | manual
  silence_threshold_db NUMERIC(5,1),
  detection_level VARCHAR(12) NOT NULL DEFAULT 'aggressive', -- gentle | standard | aggressive
  silence_min_duration_ms INTEGER NOT NULL DEFAULT 500,
  padding_before_ms INTEGER NOT NULL DEFAULT 50,
  padding_after_ms INTEGER NOT NULL DEFAULT 150,
  speech_min_duration_ms INTEGER NOT NULL DEFAULT 300,
  fillers_enabled BOOLEAN NOT NULL DEFAULT TRUE,
  repetitions_enabled BOOLEAN NOT NULL DEFAULT TRUE,
  false_starts_enabled BOOLEAN NOT NULL DEFAULT FALSE,
  sponsors_action decision_action NOT NULL DEFAULT 'suggest',
  ads_action decision_action NOT NULL DEFAULT 'suggest',
  language VARCHAR(10),                                      -- NULL = détection auto
  settings JSONB NOT NULL DEFAULT '{}'::jsonb,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE processing_step (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  media_file_id UUID NOT NULL REFERENCES media_file(id) ON DELETE CASCADE,
  kind step_kind NOT NULL,
  status step_status NOT NULL DEFAULT 'pending',
  attempt INTEGER NOT NULL DEFAULT 0,
  input_hash VARCHAR(64),                  -- empreinte des paramètres utiles
  progress NUMERIC(5,2) NOT NULL DEFAULT 0,
  error_message TEXT,
  queued_at TIMESTAMPTZ, started_at TIMESTAMPTZ, completed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (media_file_id, kind)
);

CREATE TABLE transcript (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  media_file_id UUID NOT NULL UNIQUE REFERENCES media_file(id) ON DELETE CASCADE,
  language VARCHAR(10),
  text TEXT,
  provider VARCHAR(30) NOT NULL DEFAULT 'assemblyai',
  provider_job_id VARCHAR(100),
  status step_status NOT NULL DEFAULT 'pending',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE transcript_word (        -- jamais modifié après import
  id BIGSERIAL PRIMARY KEY,
  transcript_id UUID NOT NULL REFERENCES transcript(id) ON DELETE CASCADE,
  word_index INTEGER NOT NULL,
  text TEXT NOT NULL,
  source_start_ms BIGINT NOT NULL,
  source_end_ms BIGINT NOT NULL,
  confidence NUMERIC(5,4),
  speaker VARCHAR(10),                -- A, B… si diarisation
  CHECK (source_start_ms >= 0 AND source_end_ms >= source_start_ms),
  UNIQUE (transcript_id, word_index)
);
CREATE INDEX idx_transcript_word_time ON transcript_word(transcript_id, source_start_ms);

CREATE TABLE detection (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  media_file_id UUID NOT NULL REFERENCES media_file(id) ON DELETE CASCADE,
  kind detection_kind NOT NULL,
  source_start_ms BIGINT NOT NULL,
  source_end_ms BIGINT NOT NULL,
  confidence NUMERIC(5,4),
  detector VARCHAR(60) NOT NULL,      -- ex. ffmpeg.silencedetect@1, rules.fr@1, llm.sponsor@1
  first_word_index INTEGER, last_word_index INTEGER,   -- pour les détections issues du texte
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,          -- ex. mot détecté, raison LLM
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (source_start_ms >= 0 AND source_end_ms > source_start_ms)
);
CREATE INDEX idx_detection_media_time ON detection(media_file_id, source_start_ms);
CREATE INDEX idx_detection_media_kind ON detection(media_file_id, kind);

-- décision de l'utilisateur, rattachée par recouvrement de plage (voir section Timeline)
CREATE TABLE user_override (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  media_file_id UUID NOT NULL REFERENCES media_file(id) ON DELETE CASCADE,
  kind detection_kind,                -- NULL = vaut pour toute détection recouverte, ou coupe manuelle
  action decision_action NOT NULL,    -- keep | remove
  source_start_ms BIGINT NOT NULL,
  source_end_ms BIGINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (action IN ('keep','remove') AND source_end_ms > source_start_ms)
);
CREATE INDEX idx_override_media ON user_override(media_file_id, source_start_ms);

CREATE TABLE timeline (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL REFERENCES project(id) ON DELETE CASCADE,
  version INTEGER NOT NULL,
  status timeline_status NOT NULL DEFAULT 'draft',
  fingerprint VARCHAR(64) NOT NULL,
  duration_ms BIGINT NOT NULL,
  validated_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (project_id, version),
  UNIQUE (project_id, fingerprint)
);

CREATE TABLE timeline_segment (       -- segments KEEP
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  timeline_id UUID NOT NULL REFERENCES timeline(id) ON DELETE CASCADE,
  segment_index INTEGER NOT NULL,
  media_file_id UUID NOT NULL REFERENCES media_file(id) ON DELETE CASCADE,
  source_start_ms BIGINT NOT NULL,
  source_end_ms BIGINT NOT NULL,
  output_start_ms BIGINT NOT NULL,
  output_end_ms BIGINT NOT NULL,
  removed_before_reason cut_reason,
  CHECK (source_end_ms > source_start_ms AND output_end_ms > output_start_ms),
  UNIQUE (timeline_id, segment_index)
);
CREATE INDEX idx_segment_source ON timeline_segment(media_file_id, source_start_ms);

CREATE TABLE timeline_decision (      -- pourquoi chaque détection est retirée, suggérée ou gardée
  timeline_id UUID NOT NULL REFERENCES timeline(id) ON DELETE CASCADE,
  detection_id UUID NOT NULL REFERENCES detection(id) ON DELETE CASCADE,
  action decision_action NOT NULL,
  origin decision_origin NOT NULL,
  reason TEXT,
  PRIMARY KEY (timeline_id, detection_id)
);

CREATE TABLE clip (                   -- défini en SourceTime
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL REFERENCES project(id) ON DELETE CASCADE,
  media_file_id UUID NOT NULL REFERENCES media_file(id) ON DELETE CASCADE,
  source_start_ms BIGINT NOT NULL,
  source_end_ms BIGINT NOT NULL,
  origin VARCHAR(10) NOT NULL DEFAULT 'llm',   -- llm | manual
  score NUMERIC(5,4),
  hook TEXT, reason TEXT, title TEXT, description TEXT, hashtags TEXT[],
  selected BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (source_end_ms > source_start_ms),
  CHECK (score IS NULL OR score BETWEEN 0 AND 1)
);

CREATE TABLE llm_artifact (           -- chapitres, titres du projet, notes
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  media_file_id UUID NOT NULL REFERENCES media_file(id) ON DELETE CASCADE,
  kind VARCHAR(30) NOT NULL,
  payload JSONB NOT NULL,
  provider VARCHAR(30),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE subtitle_style (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID REFERENCES project(id) ON DELETE CASCADE,   -- NULL = preset global
  name VARCHAR(100) NOT NULL,
  font_family VARCHAR(255), font_size INTEGER,
  base_color VARCHAR(9), highlight_color VARCHAR(9),
  outline_color VARCHAR(9), outline_size INTEGER,
  words_per_line SMALLINT, position VARCHAR(20), uppercase BOOLEAN NOT NULL DEFAULT TRUE,
  settings JSONB NOT NULL DEFAULT '{}'::jsonb,    -- animation pop, ombre, marges, pause de coupe de ligne
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE overlay (                -- en OutputTime relatif au début du rendu
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL REFERENCES project(id) ON DELETE CASCADE,
  clip_id UUID REFERENCES clip(id) ON DELETE CASCADE,         -- NULL = tous les rendus
  kind VARCHAR(30) NOT NULL,          -- headline | logo | watermark | text
  text TEXT, storage_key TEXT,
  start_ms BIGINT NOT NULL, end_ms BIGINT NOT NULL,
  position VARCHAR(30),
  settings JSONB NOT NULL DEFAULT '{}'::jsonb,
  CHECK (start_ms >= 0 AND end_ms > start_ms)
);

CREATE TABLE output_preset (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name VARCHAR(100) NOT NULL,
  width INTEGER NOT NULL, height INTEGER NOT NULL,
  aspect_ratio aspect_ratio NOT NULL,
  crop_mode crop_mode NOT NULL DEFAULT 'none',
  video_codec VARCHAR(50) NOT NULL DEFAULT 'h264',
  audio_codec VARCHAR(50) NOT NULL DEFAULT 'aac',
  fps NUMERIC(8,3), video_bitrate BIGINT, audio_bitrate BIGINT,
  loudness_lufs NUMERIC(4,1),
  settings JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE TABLE render (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL REFERENCES project(id) ON DELETE CASCADE,
  timeline_id UUID NOT NULL REFERENCES timeline(id) ON DELETE RESTRICT,
  kind render_kind NOT NULL,
  strategy render_strategy NOT NULL DEFAULT 'full_render',
  clip_id UUID REFERENCES clip(id) ON DELETE SET NULL,
  output_preset_id UUID NOT NULL REFERENCES output_preset(id) ON DELETE RESTRICT,
  subtitle_style_id UUID REFERENCES subtitle_style(id) ON DELETE SET NULL,
  plan JSONB NOT NULL,                -- plan de rendu envoyé au worker
  fingerprint VARCHAR(64) NOT NULL,
  status step_status NOT NULL DEFAULT 'pending',
  progress NUMERIC(5,2) NOT NULL DEFAULT 0,
  storage_key TEXT, duration_ms BIGINT, size_bytes BIGINT,
  error_message TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  started_at TIMESTAMPTZ, completed_at TIMESTAMPTZ,
  UNIQUE (project_id, fingerprint)
);
CREATE INDEX idx_render_status ON render(status);
```

Le `render.plan` est la seule copie de ce que le worker a reçu : il permet de rejouer un rendu à l'identique, même si le style ou la timeline ont changé depuis.

## Timeline et validation

Le timeline engine est une fonction pure : détections, configuration et overrides en entrée, décisions et segments KEEP en sortie. Il vit dans un package Go sans dépendance (`internal/timeline`) et tourne dans le core.

### Trois états de décision

| Action | Sens | Affichage dans l'éditeur |
| --- | --- | --- |
| `remove` | La plage sera supprimée | Zone barrée, rouge |
| `suggest` | Proposée, non appliquée tant que l'utilisateur ne la confirme pas | Zone hachurée, orange |
| `keep` | Détectée mais conservée | Marqueur discret |

Par défaut, silences, mots de remplissage et répétitions sont en `remove` selon le niveau choisi ; sponsors et pubs sont en `suggest`, parce qu'une erreur y coûte cher.

### Algorithme de reconstruction

1. Charger les détections du média, la configuration et les overrides.
2. **Silences** : garder ceux d'au moins `silence_min_duration_ms`, puis raccourcir chaque plage de `padding_after_ms` au début et de `padding_before_ms` à la fin ; une plage vide est ignorée.
3. **Textes** (remplissage, répétitions, faux départs) : retenir selon le niveau de détection et la confiance, avec une marge de quelques millisecondes autour des mots.
4. **Sponsors et pubs** : appliquer `sponsors_action` et `ads_action`.
5. **Overrides** : appliquer les décisions de l'utilisateur (règle ci-dessous).
6. **Fusion** : réunir les plages à supprimer qui se chevauchent ; le complément donne les segments KEEP.
7. **Micro-segments** : un segment KEEP plus court que `speech_min_duration_ms` est supprimé (raison `micro`), deux segments séparés par moins de 100 ms sont fusionnés.
8. **Calage** : déplacer chaque bord de coupe, dans une limite de 40 ms, vers le point le plus calme du fichier `peaks`, pour ne pas rogner un mot ni produire de clic.
9. **Calcul des temps de sortie** : cumuler les durées pour remplir `output_start_ms` et `output_end_ms`.
10. **Empreinte** : hash de (détections, config, overrides, ordre des médias). Identique à celle de la version courante, rien n'est écrit.

Niveaux de détection (points de départ à calibrer sur de vraies vidéos) :

| Niveau | Seuil auto de silence | Texte retiré |
| --- | --- | --- |
| `gentle` | plancher de bruit + 6 dB | Hésitations pures (« euh », « hum ») |
| `standard` | plancher de bruit + 9 dB | + répétitions de mots consécutifs |
| `aggressive` | plancher de bruit + 12 dB | + liste étendue (« ben », « genre »), faux départs si activés |

### Règle de rattachement des overrides

Un override s'applique à une détection quand les deux plages se recouvrent sur au moins 50 % de la durée de la détection et que le `kind` est identique (ou nul).

- `keep` sur une détection : elle passe en `keep`.
- `remove` sur une détection : elle passe en `remove`, même si le niveau ne la retenait pas, et même pour un `suggest`.
- `remove` sans détection en dessous : coupe manuelle, raison `manual`.
- `keep` sans `kind` : protège la plage, aucune suppression ne peut la toucher.

Changer le seuil de silence ou le niveau fait varier les plages détectées ; le recouvrement permet de conserver les décisions de l'utilisateur au lieu de les perdre.

### Cycle de vie et validation

- Une reconstruction crée une version `draft`, la précédente devient `superseded` (sauf si elle est `validated`).
- **Valider** fige la version : segments, décisions et durée ne changent plus. Seule une version `validated` autorise un rendu ou un export.
- Modifier la config ou un override après validation crée une nouvelle version `draft` ; la version validée reste utilisable pour les rendus déjà lancés.
- Rétention : conserver les versions validées et les cinq dernières versions `draft`.

### Calcul synchrone pour l'éditeur

Changer un curseur (durée minimale, padding, niveau) appelle `POST /timeline/preview`. Le core exécute le même calcul sur les détections déjà en base et renvoie segments et décisions sans rien écrire, en quelques dizaines de millisecondes. Le front n'a donc aucune copie de l'algorithme.

Seuls le seuil en dB et le mode auto/manuel imposent de relancer le job `silence`, parce que les silences bruts dépendent du seuil.

## Détection

Chaque détecteur produit des lignes `detection` en SourceTime avec une confiance et le nom du détecteur ; aucun ne supprime quoi que ce soit.

| Détecteur | Worker | Méthode | Points d'attention |
| --- | --- | --- | --- |
| Silences | worker-silence | `volumedetect` pour le plancher de bruit, puis `silencedetect` avec un plancher d'analyse de 50 ms | Les filtres produit (durée minimale, padding) s'appliquent plus tard dans la timeline |
| Mots de remplissage | worker-text | Dictionnaire par langue appliqué aux mots horodatés | « en fait » peut porter du sens : réglable par niveau, expérimental au départ |
| Répétitions | worker-text | Séquences consécutives de mots identiques ou quasi identiques ; la dernière occurrence est gardée | Ponctuation et casse ignorées |
| Faux départs | worker-text | Début de phrase repris presque à l'identique juste après | Heuristique, désactivé par défaut |
| Sponsors | worker-llm | LLM sur le transcript horodaté, sortie JSON validée, bornes calées sur les mots | Toujours en `suggest` par défaut |
| Pubs | worker-scenes | PySceneDetect (densité de coupes), `blackdetect` et `silencedetect` autour des césures | Surtout utile pour des enregistrements TV |
| Chapitres | worker-llm | Segmentation thématique du transcript | Produit un `llm_artifact`, pas une détection |

### Silences

Le worker lit la configuration du média :

- `threshold_mode = auto` : mesure du plancher de bruit, seuil = plancher + décalage du niveau choisi ; il renvoie `noiseFloorDb` et `calculatedThresholdDb` pour affichage.
- `manual` : seuil `silence_threshold_db` fourni par l'utilisateur.

Une heure d'audio se traite en quelques secondes. Ne détecter que sur l'audio (Opus extrait), jamais sur la vidéo.

### Texte

Le worker `text` ne lit que `words.json`. Il normalise les mots (minuscules, sans ponctuation) puis applique les règles de la langue. Les dictionnaires vivent dans des fichiers versionnés (`rules.fr@1`), le numéro de version étant enregistré dans `detection.detector`.

Point à vérifier avant de coder : AssemblyAI retire les hésitations par défaut et propose une option pour les conserver. Il faut confirmer dans sa documentation que cette option couvre le français. Sinon, une partie des « euh » n'existera tout simplement pas dans la transcription et devra être détectée côté audio.

### Sponsors, clips et chapitres (LLM)

- Le transcript est formaté en lignes horodatées compactes, découpé en blocs sémantiques si la vidéo est longue.
- La réponse est un JSON strict ; toute sortie invalide est rejetée puis rejouée une fois.
- Un `BoundaryResolver` recale chaque début et fin sur des mots, de préférence sur des fins de phrase, déduplique et classe.
- Les clips sont notés sur quatre critères : accroche, autonomie, émotion, chute. Le score et la raison s'affichent à l'utilisateur.
- Le fournisseur (OpenAI, DeepSeek, autre) se choisit par configuration ; le worker ne connaît qu'une interface.

### Pubs et scènes

PySceneDetect ne détecte que des changements de plan. Une pub est un candidat quand trois signaux se cumulent sur un segment : forte densité de coupes, image noire et silence aux bords, durée plausible. Le worker écrit `adCandidatesKey` avec un score par signal ; le core crée les détections `ad` au-dessus d'un seuil. Il travaille sur le proxy 720p, ce qui suffit et va plus vite.

## Sous-titres animés

Les sous-titres sont un fichier ASS généré par le core à partir des mots, de la timeline et du style ; le mot prononcé change de couleur, et libass le rend à l'identique dans FFmpeg et dans le navigateur.

```text
words (SourceTime) + timeline (Source→Output) + subtitle_style
        → remapping → regroupement en lignes → événements ASS
        → subtitles.ass (OutputTime)
```

### Génération

1. **Remapping** : chaque mot est converti en OutputTime par les segments KEEP. Un mot entièrement dans une zone supprimée disparaît ; un mot à cheval sur une coupe est tronqué au bord du segment.
2. **Lignes** : regrouper les mots par `words_per_line` (2 à 4), en coupant de préférence sur une ponctuation ou une pause supérieure à un seuil réglable (350 ms par défaut), avec une limite de caractères par ligne.
3. **Événements** : un événement `Dialogue` par mot. Pendant ce mot, la ligne entière est affichée, le mot actif en `highlight_color`, les autres en `base_color`. L'événement dure jusqu'au début du mot suivant, pour éviter tout clignotement entre deux mots.
4. **Dernier mot** de la ligne : prolongé d'environ 120 ms.
5. **Animation** : le mot actif peut grossir brièvement (transformation `\t` sur l'échelle, environ 110 % puis retour à 100 %).

Le style complet vient de `subtitle_style` :

| Réglage | Effet |
| --- | --- |
| Police et taille | Polices embarquées uniquement |
| Couleur de base, couleur de surbrillance | Ex. blanc et vert ou jaune, comme la vidéo de référence |
| Contour, ombre | Lisibilité sur fond variable |
| Mots par ligne, majuscules | 2 à 4, activé par défaut |
| Position, marge verticale | Bas, milieu ou haut ; marge adaptée au format 9:16 |
| Animation du mot actif | Aucune, pop, rebond léger |

### Polices

Le dossier de polices est passé à libass via `fontsdir`, avec uniquement des polices sous licence libre (famille OFL : Montserrat, Poppins, Anton, par exemple). Une police ajoutée par l'utilisateur est vérifiée avant usage. Les emojis ne sont pas rendus par libass : ils sont filtrés ou remplacés.

### Aperçu exact dans le navigateur

L'éditeur utilise JASSUB (libass compilé en WebAssembly) pour dessiner l'ASS sur la vidéo de proxy. Le rendu étant le même moteur que dans FFmpeg, ce que tu vois dans l'aperçu est ce que tu obtiens. `GET /api/projects/:id/timeline/subtitles.ass?styleId=…` sert le fichier pour une version de timeline.

### Autres formats

- **SRT** : généré depuis les mêmes lignes, sans couleur ni animation.
- **ASS brut** : exportable à côté de la timeline FCPXML, car les styles par mot ne passent pas de façon fiable dans une timeline XML.
- Si l'animation devient insuffisante, un moteur par calques (Remotion) peut être ajouté plus tard comme second renderer ; le plan de rendu l'autorise sans toucher au reste.

## Rendu et exports

Le core construit un plan de rendu complet à partir d'une timeline validée ; `worker-render` l'exécute avec FFmpeg, sans accès à la base et sans prendre de décision.

### Plan de rendu

```json
{
  "renderId": "…",
  "kind": "clip",
  "segments": [
    { "mediaKey": "media/…/source/original.mp4", "srcStartMs": 134200, "srcEndMs": 141800 }
  ],
  "crop": { "mode": "face_track", "pathKey": "media/…/reframe/crop_path.json" },
  "output": { "width": 1080, "height": 1920, "fps": 30, "codec": "h264", "crf": 20, "preset": "veryfast" },
  "audio": { "codec": "aac", "loudnessLufs": -14, "fadeMs": 15 },
  "subtitles": { "assKey": "projects/…/subtitles/….ass", "fontsDir": "/fonts" },
  "overlays": [ { "kind": "headline", "text": "…", "startMs": 0, "endMs": 3000 } ],
  "outputKey": "projects/…/renders/…/out.mp4"
}
```

### Un seul graphe FFmpeg

```text
SOURCE → trim/atrim par segment → concat → crop → scale → ASS → overlays → ENCODE → FINAL.MP4
```

Le graphe est écrit dans un fichier passé avec `-filter_complex_script`, pour éviter la limite de longueur de ligne de commande. L'audio de chaque segment reçoit un fondu d'entrée et de sortie de 10 à 20 ms, pour supprimer les clics aux coupes.

### Beaucoup de segments

Une heure de contenu coupé finement peut produire plusieurs centaines de segments, et un graphe de cette taille ralentit FFmpeg. Au-delà d'un seuil (environ 200 segments, à mesurer), le worker découpe le rendu en blocs :

- chaque bloc est encodé une seule fois, avec une keyframe forcée à son début ;
- les blocs sont ensuite assemblés avec le demuxer `concat` en copie de flux, sans ré-encodage ;
- les blocs peuvent s'encoder en parallèle si plusieurs workers sont disponibles.

La vidéo reste ainsi encodée une seule fois.

### Recadrage

| Mode | Comportement |
| --- | --- |
| `none` | Pas de crop, mise à l'échelle seule |
| `center`, `left`, `right`, `custom` | Crop fixe 9:16 ou autre ratio |
| `fit_blur` | Vidéo entière au centre sur un fond flouté |
| `face_track` | Crop suivant le visage, trajectoire lissée issue de `worker-reframe` |
| `speaker_track` | Comme `face_track`, avec bascule sur la personne qui parle (diarisation) |

`worker-reframe` ne produit qu'une trajectoire par plage de temps (centre, zoom) ; c'est `worker-render` qui l'applique. Le lissage évite les mouvements nerveux, et un changement de personne se fait par une coupe franche plutôt que par un glissement.

### Clips

Un clip est défini en SourceTime ; son rendu applique les segments KEEP de la timeline qui recoupent sa plage. Il reçoit son propre bandeau d'accroche, son propre crop et ses propres sous-titres remappés, donc il est le plus souvent rendu séparément.

L'extraction en copie de flux depuis un master (`-ss … -c copy`) reste une optimisation : elle ne s'applique que si le clip a exactement le format, le style, le crop et les overlays du master, et que les keyframes ont été forcées à son début. Un clip de 40 secondes se ré-encode en quelques secondes, donc ce chemin n'est pas prioritaire.

### Audio et encodage

- Normalisation `loudnorm` à la cible du preset (par exemple -14 LUFS pour le social).
- Débruitage optionnel appliqué avant la normalisation.
- Encodage matériel (NVENC, VideoToolbox) en option pour gagner du temps ; le logiciel (`libx264`) reste la référence de qualité.
- Progression lue depuis `-progress` de FFmpeg et publiée sur `studio.progress`.

### Exports

Les exports sont des fichiers texte générés par le core à partir de la timeline validée (quasi aucun coût de calcul) et déposés dans MinIO.

| Format | Usage |
| --- | --- |
| FCPXML | Final Cut Pro, importable dans DaVinci Resolve |
| EDL (CMX 3600) | Format universel, métadonnées limitées |
| OTIO | Format ouvert, représentation de référence pour dériver les autres |
| XML Premiere (`xmeml`) | Premiere Pro, format distinct de FCPXML |
| Dossier de clips numérotés | Import CapCut : 001, 002, 003 dans l'ordre |
| SRT et ASS | À côté de la timeline, car les styles par mot ne passent pas en XML |
| MP3 | Audio seul, rendu par `worker-render` |

## API HTTP

L'API Go/Fiber expose des commandes synchrones qui écrivent en base et dans l'outbox, et des requêtes de lecture ; elle ne fait aucun traitement média.

Conventions : JSON en camelCase, authentification `Authorization: Bearer <clé>`, erreurs `{ "message", "errors?", "code?" }`, pagination par curseur, en-tête `Idempotency-Key` accepté sur les créations. Le contrat OpenAPI est maintenu dans `docs/openapi.yaml`, et le front en génère ses types.

| Ressource | Endpoints |
| --- | --- |
| Santé | `GET /livez`, `/readyz`, `/startupz` |
| Projets | `GET, POST /api/projects` ; `GET, PATCH, DELETE /api/projects/:id` |
| Upload | `POST /api/projects/:id/media/uploads` (initie le multipart), `POST /api/media/:id/uploads/parts` (URLs présignées par lot), `POST /api/media/:id/uploads/complete` |
| Lien | `POST /api/projects/:id/media/from-url` |
| Médias | `GET /api/projects/:id/media`, `GET, DELETE /api/media/:id` (avec l'état de chaque étape), `PATCH /api/media/:id/configuration`, `POST /api/media/:id/reanalyze` (étapes à relancer), `POST /api/media/:id/steps/:step/retry` |
| Transcript | `GET /api/media/:id/transcript?fromMs=&toMs=`, `GET /api/media/:id/transcript.srt` |
| Détections | `GET /api/media/:id/detections?kind=&fromMs=&toMs=` |
| Overrides | `POST /api/media/:id/overrides` (keep, remove, coupe manuelle), `DELETE /api/overrides/:id` |
| Timeline | `GET /api/projects/:id/timeline`, `GET …/timeline/versions`, `POST …/timeline/preview` (calcul sans écriture), `POST …/timeline/validate` |
| Sous-titres | `GET …/timeline/subtitles.ass` et `.srt` (paramètre `styleId`), CRUD `/api/subtitle-styles` |
| Overlays | CRUD `/api/projects/:id/overlays` |
| Clips | `GET /api/projects/:id/clips`, `PATCH /api/clips/:id` (sélection, bornes), `POST …/clips` (clip manuel), `POST …/clips/analyze` |
| Rendus | `POST /api/projects/:id/renders`, `GET /api/projects/:id/renders`, `GET /api/renders/:id`, `POST /api/renders/:id/cancel`, `GET /api/renders/:id/download` |
| Exports | `POST /api/projects/:id/exports` (format, version), `GET /api/exports/:id/download` |
| Presets | `GET /api/output-presets` |
| Temps réel | `GET /api/realtime/connection` |

Règles utiles :

- `POST …/timeline/validate` exige la version en paramètre ; si une version plus récente existe, la requête échoue avec `409`.
- `POST …/renders` refuse une timeline non validée (`422`) et renvoie le rendu existant si l'empreinte est identique.
- `reanalyze` ne relance que les étapes dont l'`input_hash` a changé, sauf demande explicite ; la transcription n'est jamais relancée implicitement, car elle est payante.
- Les téléchargements renvoient une URL présignée MinIO de courte durée.

## Temps réel

Centrifugo pousse vers le navigateur des messages courts qui disent quoi recharger ; l'état reste dans l'API, jamais dans le message.

Le front se connecte avec le jeton renvoyé par `GET /api/realtime/connection` et s'abonne au canal `projects:{projectId}`. Les types suivent la convention `entité.action`, comme dans Uptime.

| Type | Quand | Payload |
| --- | --- | --- |
| `media.step_started` | Un worker démarre une étape | mediaId, step |
| `media.step_progress` | Progression (éphémère, issue de `q.progress`) | mediaId, step, percent |
| `media.step_completed`, `media.step_failed` | Fin d'étape, résultat persisté | mediaId, step, erreur éventuelle |
| `transcript.completed` | Mots importés | mediaId |
| `detections.updated` | Nouvelles détections | mediaId, kind |
| `timeline.rebuilt` | Nouvelle version `draft` | projectId, version |
| `timeline.validated` | Version figée | projectId, version |
| `clips.updated` | Candidats ajoutés ou modifiés | projectId |
| `render.progress` | Avancement d'un rendu | renderId, percent |
| `render.completed`, `render.failed` | Fin de rendu | renderId |

Côté front, un message déclenche l'invalidation de la requête TanStack Query concernée, qui recharge les données. Une reconnexion invalide tout le projet, ce qui rend un message perdu sans conséquence.

## Front Next.js

Le front est une application Next.js (App Router, TypeScript) dont l'écran central est un éditeur de timeline qui joue la vidéo en sautant les coupes, sans aucun rendu.

### Stack

| Besoin | Choix |
| --- | --- |
| UI | Tailwind CSS + shadcn/ui |
| Données serveur | TanStack Query, types générés depuis `openapi.yaml` |
| État de l'éditeur | Zustand (tête de lecture, zoom, sélection, filtres) |
| Temps réel | centrifuge-js |
| Sous-titres à l'écran | JASSUB (libass en WebAssembly) |
| Timeline et waveform | Canvas 2D maison, virtualisé |
| Upload | Multipart direct vers MinIO par URL présignée, parties en parallèle |
| Formulaires | React Hook Form + Zod |

### Écrans

| Route | Contenu |
| --- | --- |
| `/` | Liste des projets, création |
| `/projects/[id]` | Médias du projet, ajout par upload ou lien, état de chaque étape en direct |
| `/projects/[id]/edit` | Éditeur : lecteur, timeline, transcript, détections, réglages |
| `/projects/[id]/subtitles` | Styles de sous-titres avec aperçu en direct |
| `/projects/[id]/clips` | Clips candidats : score, accroche, raison, aperçu, sélection, bornes |
| `/projects/[id]/renders` | Rendus et exports, progression, téléchargement |
| `/settings` | Presets de sortie, polices, valeurs par défaut |

### Éditeur de timeline

L'éditeur affiche, de haut en bas :

- le **lecteur** (proxy 720p) avec les sous-titres JASSUB par-dessus ;
- la **timeline** : waveform, détections colorées par famille, segments KEEP, tête de lecture, zoom à la molette ;
- le **transcript** : mots cliquables, mots supprimés barrés, mot en cours mis en évidence ;
- le **panneau de détections** : liste filtrable par famille, compteurs, temps gagné, boutons « tout garder » et « tout supprimer » par famille ;
- le **panneau de réglages** : seuil, durée minimale, padding, niveau, avec recalcul en direct.

Interactions principales :

| Action | Effet |
| --- | --- |
| Clic sur une coupe | Sélection, lecture de la coupe avec 1 seconde de contexte avant et après |
| Entrée ou bouton | Bascule garder ou supprimer, crée ou supprime un `user_override` |
| Sélection dans la waveform ou le texte | Crée une coupe manuelle |
| Curseur de réglage | Appel de `POST /timeline/preview` (délai de 150 ms), mise à jour de l'affichage |
| Valider | `POST /timeline/validate`, la version devient lecture seule |

Les actions d'édition s'empilent dans un journal local qui permet l'annulation et le rétablissement ; annuler supprime l'override correspondant.

### Lecture avec sauts de coupes

Le lecteur joue le proxy en suivant la liste des segments KEEP : une boucle sur l'horloge de la vidéo détecte la fin du segment courant et déplace la lecture au début du suivant. Le proxy a une keyframe toutes les 0,5 s pour que les sauts restent rapides. De petits à-coups restent possibles dans l'aperçu, et le rendu final n'en a pas.

### Upload

Le front découpe le fichier en parties, demande les URLs présignées par lot, envoie jusqu'à quatre parties en parallèle et reprend aux parties manquantes après une coupure réseau. L'API ne voit jamais les octets de la vidéo.

## Stockage et ingestion

MinIO contient tous les fichiers ; la base ne garde que des clés et des métadonnées, et les messages ne transportent que des clés.

```text
bucket studio
  media/{mediaId}/source/original.<ext>
  media/{mediaId}/audio/audio.opus
  media/{mediaId}/proxy/proxy_720p.mp4
  media/{mediaId}/peaks/peaks.bin
  media/{mediaId}/transcript/words.json
  media/{mediaId}/transcript/subtitles.srt
  media/{mediaId}/transcript/provider.job
  media/{mediaId}/analysis/silences.json | text.json | llm.json | scenes.json
  media/{mediaId}/reframe/crop_path.json
  projects/{projectId}/subtitles/{timelineVersion}/{styleId}.ass
  projects/{projectId}/renders/{renderId}/out.mp4
  projects/{projectId}/exports/{timelineVersion}/timeline.fcpxml | .edl | .otio
```

### Upload par fichier

1. `POST …/media/uploads` crée le média (`pending`) et un upload multipart S3.
2. Le navigateur demande les URLs présignées des parties, puis envoie directement à MinIO.
3. `POST …/uploads/complete` assemble l'objet, vérifie la taille, passe le média à `processing` et publie `media.uploaded`.

Un upload jamais terminé est purgé après 24 heures par une tâche de nettoyage (abort du multipart).

### Ajout par lien

`POST …/media/from-url` crée le média en `fetching` et publie un job `fetch`. `worker-fetch` appelle yt-dlp avec une taille et une durée maximales, dépose le fichier dans `source/`, puis le core publie `media.fetched`.

- Le téléchargement par lien est fragile : yt-dlp casse régulièrement et certains sites le bloquent. Le binaire est mis à jour par image Docker, et l'upload reste le chemin de référence.
- N'utiliser que des contenus dont tu as les droits ; les conditions d'utilisation de la plupart des plateformes interdisent le téléchargement.
- Les erreurs de `fetch` sont classées (lien invalide, contenu privé, trop long) pour afficher un message clair.

### Fichiers dérivés

| Fichier | Rôle | Paramètres |
| --- | --- | --- |
| `audio.opus` | Entrée de la silence, de la transcription et du waveform | Mono, débit réduit, suffisant pour la parole |
| `proxy_720p.mp4` | Aperçu navigateur, scenes, reframe | H.264, keyframe toutes les 0,5 s, `faststart` |
| `peaks.bin` | Waveform et calage des coupes | 100 points par seconde, min et max sur un octet |

### Rétention

- Les sources et les rendus validés sont conservés jusqu'à suppression du projet.
- Les proxys, rendus intermédiaires et blocs de rendu sont purgés après 7 jours.
- Supprimer un projet supprime ses objets MinIO via un job de nettoyage, pas en ligne dans la requête.

La taille du disque est le premier facteur limitant en local : les vidéos de plusieurs gigaoctets, le proxy et les rendus s'additionnent.

## Configuration, sécurité, observabilité, tests

Le projet tourne en Docker Compose en local, comme Uptime, avec un conteneur par binaire et un `Makefile` de même forme.

### Environnement local

| Service | Port hôte | Remarque |
| --- | --- | --- |
| API | 4000 | Rechargement à chaud (Air) |
| Front Next.js | 3000 |  |
| PostgreSQL | 9543 |  |
| RabbitMQ | 5672, 15672 | AMQP et interface d'administration |
| MinIO | 9000, 9001 | API S3 et console |
| Centrifugo | 8000 | WebSocket et API HTTP |
| Workers | — | `worker`, `worker-fetch`, `worker-media`, `worker-silence`, `worker-transcribe`, `worker-text`, `worker-llm`, `worker-scenes`, `worker-reframe`, `worker-render` |

Les variables clés : `API_KEY`, `DATABASE_URL`, `RABBITMQ_URL`, `S3_ENDPOINT`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`, `S3_BUCKET`, `ASSEMBLYAI_API_KEY`, `LLM_PROVIDER`, `LLM_API_KEY`, `CENTRIFUGO_*`, `RENDER_ENCODER` (software ou hardware), `MAX_UPLOAD_BYTES`.

### Sécurité

- Clé API statique comparée en temps constant ; Clerk reste possible plus tard.
- Aucun secret dans les messages : les workers reçoivent leurs clés par environnement. Les URLs présignées sont courtes.
- `worker-fetch` limite taille, durée et protocoles, et refuse les adresses privées (protection contre les requêtes vers le réseau interne).
- Les arguments FFmpeg sont construits en liste, jamais par concaténation de chaîne ; les textes de sous-titres et de bandeaux sont échappés.
- Les polices fournies par l'utilisateur sont validées avant d'être passées à libass.
- Journaux sans contenu des transcripts.

### Observabilité

- Logs structurés avec `correlationId` (le média) et `messageId` dans chaque ligne.
- Métriques : durée par type de job, taille des queues et des DLQ, minutes transcrites, minutes rendues, taux d'échec.
- Alerte simple sur toute DLQ non vide.
- `processing_step` donne l'état de chaque étape, visible à l'écran et par API.

### Tests

| Niveau | Cible | Outil |
| --- | --- | --- |
| Unitaires | Timeline engine, remapping, règles de texte, génération ASS, exports | `go test`, tables de cas, sans infrastructure |
| Contrats | Enveloppe et payloads de chaque job | Fichiers JSON d'exemple versionnés, validés des deux côtés |
| Intégration | Repositories, outbox, flux complet d'un média | Postgres, RabbitMQ et MinIO du Compose |
| Workers | Un job de bout en bout avec un fichier court de référence | Vidéo de 10 à 20 secondes dans le dépôt |
| Handlers HTTP | Même approche qu'Uptime | Fiber `app.Test` avec mocks |
| Front | Logique de l'éditeur (segments, undo) | Vitest, Playwright pour le parcours principal |

Le timeline engine est la pièce à couvrir le plus : une table de cas (silence au bord, chevauchements, override, micro-segment, calage) vaut mieux que n'importe quel test d'intégration.

## Plan de construction

Le produit est conçu en entier, mais chaque pièce se construit et se teste seule, dans un ordre où l'étape suivante a toujours de quoi se brancher. Une étape est terminée quand son critère de fin est vérifié, pas quand le code existe.

| N° | Contenu | Critère de fin |
| --- | --- | --- |
| 0 | Socle : monorepo, Compose (Postgres, RabbitMQ, MinIO, Centrifugo), Goose, outbox, déclaration de la topologie RabbitMQ, packages communs (enveloppe, publisher avec confirms, consumer avec retry, DLQ et idempotence) | Un job de test fait API → outbox → worker → résultat → Centrifugo |
| 1 | Projets, médias, upload multipart vers MinIO, authentification par clé | Un fichier de 2 Go est uploadé, et repris après une coupure réseau |
| 2 | `worker-media` : probe, audio Opus, proxy 720p, waveform ; `processing_step` et temps réel | Un média passe à `ready`, les étapes s'affichent en direct |
| 3 | `worker-silence`, table `detection`, `media_configuration` | Les silences d'un média sont listés avec seuil auto et seuil manuel |
| 4 | Timeline engine (silences), versions, `preview`, `validate`, `SourceToOutput` | Table de tests verte ; une timeline validée est servie par l'API |
| 5 | Front : projets, upload, éditeur avec lecture à sauts de coupes, bascule garder ou supprimer | Aperçu fluide sur un média réel, décision modifiable et validée |
| 6 | `worker-transcribe`, import des mots, vue transcript | Mots en base ; un retry ne crée jamais un second job AssemblyAI |
| 7 | `worker-text` : remplissage, répétitions, faux départs ; overrides complets ; mots barrés dans l'éditeur | Un « garder » survit à un changement de seuil |
| 8 | Génération ASS, styles, aperçu JASSUB | L'aperçu et le rendu FFmpeg sont identiques à l'œil |
| 9 | `worker-render` : master 16:9, fondus, loudnorm, rendu par blocs ; exports FCPXML, EDL, OTIO, SRT, ASS | MP4 sous-titré lisible ; FCPXML importé dans Resolve ou Final Cut sans erreur |
| 10 | `worker-llm` : sponsors et chapitres, `suggest` dans l'éditeur | Un sponsor est proposé, confirmable en un clic |
| 11 | Clips : analyse virale, page clips, bornes sur les phrases, rendu 9:16 en crop centré, bandeau d'accroche | Un clip vertical sous-titré avec bandeau, rendu depuis un clic |
| 12 | `worker-reframe` : suivi de visage, bascule sur le locuteur | Le crop suit un visage sans à-coups sur une vidéo de test |
| 13 | `worker-fetch` : ajout par lien | Un lien produit un média au même état qu'un upload |
| 14 | `worker-scenes` : candidats de pubs | Une pub d'un enregistrement TV est proposée en `suggest` |
| 15 | Finitions : presets, auto-zoom, débruitage, titres et hashtags, XML Premiere, dossier numéroté, rétention, métriques | Presets appliqués en un clic, purge automatique en place |

Règles de conduite pendant la construction :

- **Contrat d'abord** : chaque nouveau job commence par son payload JSON d'exemple et son test de contrat.
- **Un fichier de référence court** (10 à 20 secondes, avec parole, silences et un « euh ») sert de test pour tous les workers.
- **Pas de worker en parallèle d'un autre non terminé** : on en démarre un nouveau quand le précédent a son critère de fin.
- **Mesurer tôt** : à l'étape 9, chronométrer un rendu d'une heure pour fixer le seuil de découpage en blocs.

## Points de vigilance et décisions ouvertes

Ces points sont les endroits où le projet peut dérailler ; plusieurs se règlent par une mesure ou une vérification de documentation, pas par de la réflexion.

### Risques techniques

| Risque | Pourquoi | Parade |
| --- | --- | --- |
| Coupes qui rognent un mot | Les timestamps ASR sont approximatifs | Calage sur l'énergie du signal, fondus de 10 à 20 ms, écoute de chaque coupe |
| Graphe FFmpeg trop gros | Des centaines de segments ralentissent le rendu | Rendu par blocs avec copie de flux à l'assemblage, seuil à mesurer |
| Messages longs et `consumer_timeout` | RabbitMQ ferme le canal d'un worker trop lent | Délai porté à 2 heures, rendus découpés en blocs |
| Aperçu saccadé | Le navigateur saute entre segments | Proxy avec keyframes fréquentes ; le rendu final n'est pas concerné |
| Seuil de silence mal calibré | Le plancher de bruit varie selon la source | Mode auto avec décalage par niveau, valeurs de départ à calibrer sur de vraies vidéos |
| Doublon de transcription payante | At-least-once et retries | Marqueur `provider.job` dans MinIO, jamais de relance implicite |
| Empreinte de timeline identique à une ancienne version | Un retour à une config passée recréerait la même empreinte | Si l'empreinte existe, la version correspondante redevient la tête au lieu d'être recréée |
| Disque saturé | Sources, proxys et rendus s'additionnent | Rétention de 7 jours sur les dérivés, surveillance de l'espace libre |

### À vérifier avant de coder

- **AssemblyAI** : support de la conservation des hésitations en français, de la diarisation et des mots horodatés à la milliseconde pour la langue visée ; tarif par minute et limites de taille.
- **Version de RabbitMQ** : réglage du `consumer_timeout` par queue, comportement du dead-lettering « at-least-once » sur les quorum queues.
- **Licences** : Ultralytics YOLO est sous AGPL-3.0 (MediaPipe suffit pour un visage), Remotion demande une licence au-delà d'une certaine taille d'organisation ; polices sous OFL uniquement.
- **yt-dlp** : stabilité sur les plateformes visées ; l'upload reste le chemin de référence.
- **Droits des contenus** : le clipping de vidéos de tiers expose à des retraits et à des problèmes de monétisation ; travailler sur du contenu dont tu as les droits.

### Décisions ouvertes

| Décision | Options | Recommandation |
| --- | --- | --- |
| Fournisseur LLM par défaut | OpenAI, DeepSeek, autre | Interface abstraite, choisir après un essai sur 3 vidéos |
| Encodage du rendu | Logiciel (`libx264`) ou matériel | Logiciel par défaut, matériel en option |
| Format du fichier de crop | JSON par plage de temps ou courbe échantillonnée | Plages avec interpolation, plus compact |
| Un projet pour plusieurs médias | Timeline concaténée dans l'ordre | Oui, via `media_file.position` |
| Langue des règles de texte | Français d'abord | Français, anglais ensuite, étiqueté expérimental |

### Mesures à faire dès que possible

- Durée de la transcription AssemblyAI par minute d'audio, sur un fichier réel.
- Durée d'un rendu 1080x1920 d'une heure, avec 50 puis 500 segments.
- Précision des coupes sur une dizaine de vidéos de nature différente (podcast, vlog, gameplay).

Leçon du projet précédent : le coût se joue sur le volume traité, il faut donc le mesurer sur des vidéos représentatives avant de fixer des paramètres par défaut.
