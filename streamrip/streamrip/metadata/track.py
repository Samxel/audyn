from __future__ import annotations

import logging
from dataclasses import dataclass
from typing import Optional

from .album import AlbumMetadata
from .util import typed

logger = logging.getLogger("streamrip")


@dataclass(slots=True)
class TrackInfo:
    id: str
    quality: int

    bit_depth: Optional[int] = None
    explicit: bool = False
    sampling_rate: Optional[int | float] = None
    work: Optional[str] = None


@dataclass(slots=True)
class TrackMetadata:
    info: TrackInfo

    title: str
    album: AlbumMetadata
    artist: str
    tracknumber: int
    discnumber: int
    composer: str | None
    isrc: str | None = None
    lyrics: str | None = ""

    @classmethod
    def from_deezer(cls, album: AlbumMetadata, resp) -> TrackMetadata | None:
        track_id = str(resp["id"])
        isrc = typed(resp["isrc"], str)
        bit_depth = 16
        sampling_rate = 44.1
        explicit = typed(resp["explicit_lyrics"], bool)
        title = typed(resp["title"], str)
        artist = typed(resp["artist"]["name"], str)
        tracknumber = typed(resp["track_position"], int)
        discnumber = typed(resp["disk_number"], int)
        info = TrackInfo(
            id=track_id,
            quality=album.info.quality,
            bit_depth=bit_depth,
            explicit=explicit,
            sampling_rate=sampling_rate,
            work=None,
        )
        return cls(
            info=info,
            title=title,
            album=album,
            artist=artist,
            tracknumber=tracknumber,
            discnumber=discnumber,
            composer=None,
            isrc=isrc,
        )

    @classmethod
    def from_resp(cls, album: AlbumMetadata, source: str, resp) -> TrackMetadata | None:
        if source != "deezer":
            raise Exception(f"Unsupported source: {source}")
        return cls.from_deezer(album, resp)

    def format_track_path(self, format_string: str) -> str:
        none_text = "Unknown"
        info = {
            "id": self.info.id,
            "title": self.title,
            "tracknumber": self.tracknumber,
            "artist": self.artist,
            "albumartist": self.album.albumartist,
            "albumcomposer": self.album.albumcomposer or none_text,
            "composer": self.composer or none_text,
            "explicit": " (Explicit) " if self.info.explicit else "",
        }
        return format_string.format(**info)
