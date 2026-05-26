import functools
from typing import Optional, Type, TypeVar


def get_album_track_ids(source: str, resp) -> list[str]:
    """Return the list of track IDs from an album API response."""
    tracklist = resp["tracks"]
    return [str(track["id"]) for track in tracklist]


def safe_get(dictionary, *keys, default=None):
    return functools.reduce(
        lambda d, key: d.get(key, default) if isinstance(d, dict) else default,
        keys,
        dictionary,
    )


T = TypeVar("T")


def typed(thing, expected_type: Type[T]) -> T:
    assert isinstance(thing, expected_type)
    return thing


def get_quality_id(
    bit_depth: Optional[int],
    sampling_rate: Optional[int | float],
) -> int:
    """Get the universal quality id from bit depth and sampling rate."""
    if bit_depth is None or sampling_rate is None:
        return 1
    if bit_depth == 16:
        return 2
    if bit_depth == 24:
        if sampling_rate <= 96:
            return 3
        return 4
    raise Exception(f"Invalid {bit_depth = }")
