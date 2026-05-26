# Audyn

Audyn is a fake Newznab indexer and download client that integrates Deezer into the *arr stack.
It speaks Newznab to Prowlarr and SABnzbd to Lidarr, while actually searching and downloading from Deezer.

## How it works

- **Indexer**: Audyn pretends to be a Newznab usenet indexer, forwarding search queries to the Deezer API
- **Downloader**: Audyn pretends to be a SABnzbd download client, downloading tracks directly from Deezer

## Stack

- Prowlarr → Audyn (Newznab)
- Lidarr → Audyn (SABnzbd)
- Audyn → Deezer API

## Setup

### 1. Get your Deezer ARL token
Log in to [deezer.com](https://deezer.com), open DevTools → Application → Cookies and copy the value of the `arl` cookie.<br>
Full guide: [Finding Your Deezer ARL Cookie](https://github.com/nathom/streamrip/wiki/Finding-Your-Deezer-ARL-Cookie)

### 2. Configure docker-compose.yml
Clone the repo and edit `compose.yml`:

```yaml
services:
  audyn:
    build: .
    container_name: audyn
    restart: unless-stopped
    ports:
      - "5000:5000"
    environment:
      # Deezer ARL cookie
      DEEZER_ARL: "your-arl-here"
      # Quality: 0=MP3_128  1=MP3_320  2=FLAC (requires Deezer HiFi)
      # DEEZER_QUALITY: "1"
      AUDYN_PORT: "5000"
      AUDYN_INCOMPLETE_PATH: /downloads/incomplete
      AUDYN_COMPLETE_PATH: /downloads/complete
      # The path Lidarr sees (if Audyn and Lidarr mount the same volume under different paths)
      AUDYN_COMPLETE_PATH_MAPPING: /downloads/complete
    volumes:
      - /your/downloads/path:/downloads
```

Make sure the download path matches what Lidarr has mounted as its download folder.

### 3. Start Audyn
```bash
docker compose up -d
```

### 4. Add Audyn to Lidarr

**As an indexer** (Settings → Indexers → Add → Generic Newznab):
- URL: `http://audyn-host:5000`
- API Path: `/api`
- API Key: anything, e.g. `audyn`

**As a download client** (Settings → Download Clients → Add → SABnzbd):
- Host: `audyn-host`
- Port: `5000`
- URL Base: `download`
- API Key: anything, e.g. `audyn`

### 5. Lidarr settings

**Settings → Media Management:**
- Delete empty folders: true
- Allow Fingerprinting: Always

**Settings → Quality:**
- Increase the maximum file size for FLAC - downloads get large quickly and Lidarr will reject them otherwise.

**Permissions:**
- Audyn creates folders with `0o755`
- Set chmod Folder to `755` in Lidarr under Settings → Media Management

## Troubleshooting

**"One or more tracks expected in this release were not imported or missing from the release"**

This usually means Lidarr couldn't match the Deezer release to a MusicBrainz entry closely enough which is common with less known artists.<br>
Go to **Wanted → Manual Import**, navigate to the download folder and manually assign the correct album.

## Credits

Downloads powered by [streamrip](https://github.com/nathom/streamrip) by nathom (stripped down to Deezer-only for this project)

## Disclaimer

This project is provided for educational purposes only.
By using Audyn, you agree to the terms and conditions of the Deezer API. The author is not responsible for any misuse of this software or any violations of Deezer's terms of service. Use at your own risk.

This project was developed with the assistance of AI tools (Claude by Anthropic).
