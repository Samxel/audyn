from __future__ import annotations

import logging
from dataclasses import dataclass

logger = logging.getLogger("streamrip")


@dataclass(slots=True)
class ArtistMetadata:
    name: str
    ids: list[str]

    def album_ids(self):
        return self.ids

    @classmethod
    def from_resp(cls, resp: dict, source: str) -> ArtistMetadata:
        if source != "deezer":
            raise NotImplementedError(f"Unsupported source: {source}")
        return cls(resp["name"], [str(a["id"]) for a in resp["albums"]])
