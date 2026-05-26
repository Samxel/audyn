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

## Credits

Downloads powered by [streamrip](https://github.com/nathom/streamrip) by nathom (stripped down to Deezer-only for this project)

## Disclaimer

This project is provided for educational purposes only.
By using Audyn, you agree to the terms and conditions of the Deezer API. The author is not responsible for any misuse of this software or any violations of Deezer's terms of service. Use at your own risk.

This project was developed with the assistance of AI tools (Claude by Anthropic).
