import asyncio
import logging
from dataclasses import dataclass

from ..client import Client
from ..config import Config
from ..db import Database
from ..exceptions import NonStreamableError
from ..metadata import ArtistMetadata
from .album import PendingAlbum
from .media import Media, Pending

logger = logging.getLogger("streamrip")

# Resolve only N albums at a time to avoid initial latency
RESOLVE_CHUNK_SIZE = 10


@dataclass(slots=True)
class Artist(Media):
    """Represents all albums from a single Deezer artist."""

    name: str
    albums: list[PendingAlbum]
    client: Client
    config: Config

    async def preprocess(self):
        pass

    async def download(self):
        await self._download_async()

    async def postprocess(self):
        pass

    async def _download_async(self):
        async def _rip(item: PendingAlbum):
            album = await item.resolve()
            if album is None:
                return
            await album.rip()

        batches = self.batch(
            [_rip(album) for album in self.albums],
            RESOLVE_CHUNK_SIZE,
        )
        for batch in batches:
            await asyncio.gather(*batch)

    @staticmethod
    def batch(iterable, n=1):
        total = len(iterable)
        for ndx in range(0, total, n):
            yield iterable[ndx : min(ndx + n, total)]


@dataclass(slots=True)
class PendingArtist(Pending):
    id: str
    client: Client
    config: Config
    db: Database

    async def resolve(self) -> Artist | None:
        try:
            resp = await self.client.get_metadata(self.id, "artist")
        except NonStreamableError as e:
            logger.error(
                f"Artist {self.id} not available to stream on {self.client.source} ({e})",
            )
            return None

        try:
            meta = ArtistMetadata.from_resp(resp, self.client.source)
        except Exception as e:
            logger.error(f"Error building artist metadata: {e}")
            return None

        albums = [
            PendingAlbum(album_id, self.client, self.config, self.db)
            for album_id in meta.album_ids()
        ]
        return Artist(meta.name, albums, self.client, self.config)
