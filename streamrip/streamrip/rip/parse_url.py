from __future__ import annotations

import logging
import re
from abc import ABC, abstractmethod

from ..client import Client
from ..config import Config
from ..db import Database
from ..media import (
    Pending,
    PendingAlbum,
    PendingArtist,
    PendingLabel,
    PendingPlaylist,
    PendingSingle,
)

logger = logging.getLogger("streamrip")

# Matches standard Deezer URLs: https://www.deezer.com/album/12345
DEEZER_URL_REGEX = re.compile(
    r"https?://(?:www|open|play|listen)?\.?deezer\.com?(?:(?:/(album|artist|track|playlist|video|label))|(?:\/[-\w]+?))+\/([-\w]+)",
)


class URL(ABC):
    match: re.Match
    source: str

    def __init__(self, match: re.Match, source: str):
        self.match = match
        self.source = source

    @classmethod
    @abstractmethod
    def from_str(cls, url: str) -> URL | None:
        raise NotImplementedError

    @abstractmethod
    async def into_pending(
        self,
        client: Client,
        config: Config,
        db: Database,
    ) -> Pending:
        raise NotImplementedError


class DeezerURL(URL):
    """Handles standard Deezer URLs like https://www.deezer.com/album/12345."""

    @classmethod
    def from_str(cls, url: str) -> URL | None:
        m = DEEZER_URL_REGEX.match(url)
        if m is None:
            return None
        media_type, item_id = m.groups()
        if media_type is None or item_id is None:
            return None
        return cls(m, "deezer")

    async def into_pending(
        self,
        client: Client,
        config: Config,
        db: Database,
    ) -> Pending:
        media_type, item_id = self.match.groups()
        assert client.source == "deezer"

        if media_type == "track":
            return PendingSingle(item_id, client, config, db)
        elif media_type == "album":
            return PendingAlbum(item_id, client, config, db)
        elif media_type == "playlist":
            return PendingPlaylist(item_id, client, config, db)
        elif media_type == "artist":
            return PendingArtist(item_id, client, config, db)
        elif media_type == "label":
            return PendingLabel(item_id, client, config, db)
        raise NotImplementedError(f"Unsupported media type: {media_type}")


class DeezerDynamicURL(URL):
    """Handles dynamic/short Deezer links like https://deezer.page.link/xxx."""

    standard_link_re = re.compile(
        r"https://www\.deezer\.com/[a-z]{2}/(album|artist|playlist|track)/(\d+)"
    )
    dynamic_link_re = re.compile(r"https://(?:deezer|dzr)\.page\.link/\w+")

    @classmethod
    def from_str(cls, url: str) -> URL | None:
        match = cls.dynamic_link_re.match(url)
        if match is None:
            return None
        return cls(match, "deezer")

    async def into_pending(
        self,
        client: Client,
        config: Config,
        db: Database,
    ) -> Pending:
        url = self.match.group(0)  # entire dynamic link
        media_type, item_id = await self._extract_info_from_dynamic_link(url, client)
        if media_type == "track":
            return PendingSingle(item_id, client, config, db)
        elif media_type == "album":
            return PendingAlbum(item_id, client, config, db)
        elif media_type == "playlist":
            return PendingPlaylist(item_id, client, config, db)
        elif media_type == "artist":
            return PendingArtist(item_id, client, config, db)
        elif media_type == "label":
            return PendingLabel(item_id, client, config, db)
        raise NotImplementedError(f"Unsupported media type: {media_type}")

    @classmethod
    async def _extract_info_from_dynamic_link(
        cls, url: str, client: Client
    ) -> tuple[str, str]:
        async with client.session.get(url) as resp:
            match = cls.standard_link_re.search(await resp.text())

        if match:
            return match.group(1), match.group(2)

        raise Exception("Unable to extract Deezer dynamic link.")


def parse_url(url: str) -> URL | None:
    """Return a URL object for the given Deezer URL string, or None if unrecognised."""
    url = url.strip()
    for cls in (DeezerURL, DeezerDynamicURL):
        result = cls.from_str(url)
        if result is not None:
            return result
    return None
