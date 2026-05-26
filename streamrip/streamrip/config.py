"""Classes and functions that manage config state."""

import copy
import functools
import logging
import os
import shutil
from dataclasses import dataclass, fields
from pathlib import Path

import click
from tomlkit.api import dumps, parse
from tomlkit.toml_document import TOMLDocument

logger = logging.getLogger("streamrip")

APP_DIR = click.get_app_dir("streamrip")
os.makedirs(APP_DIR, exist_ok=True)
DEFAULT_CONFIG_PATH = os.path.join(APP_DIR, "config.toml")
CURRENT_CONFIG_VERSION = "2.2.0"


class OutdatedConfigError(Exception):
    pass


@dataclass(slots=True)
class DeezerConfig:
    # An authentication cookie that allows streamrip to use your Deezer account
    # See https://github.com/nathom/streamrip/wiki/Finding-Your-Deezer-ARL-Cookie
    arl: str
    # 0=MP3_128, 1=MP3_320, 2=FLAC
    quality: int
    # If the target quality is not available, fallback to best available
    lower_quality_if_not_available: bool
    # Allow free 320kbps MP3 downloads when no ARL is provided
    use_deezloader: bool
    # Warn when falling back to deezloader
    deezloader_warnings: bool


@dataclass(slots=True)
class DatabaseConfig:
    downloads_enabled: bool
    downloads_path: str
    failed_downloads_enabled: bool
    failed_downloads_path: str


@dataclass(slots=True)
class ConversionConfig:
    enabled: bool
    # FLAC, ALAC, OPUS, MP3, VORBIS, or AAC
    codec: str
    # In Hz
    sampling_rate: int
    # Only 16 and 24 are available
    bit_depth: int
    # Only applicable for lossy codecs
    lossy_bitrate: int


@dataclass(slots=True)
class ArtworkConfig:
    embed: bool
    embed_size: str
    embed_max_width: int
    save_artwork: bool
    saved_max_width: int


@dataclass(slots=True)
class MetadataConfig:
    set_playlist_to_album: bool
    renumber_playlist_tracks: bool
    exclude: list[str]


@dataclass(slots=True)
class FilepathsConfig:
    add_singles_to_folder: bool
    folder_format: str
    track_format: str
    restrict_characters: bool
    truncate_to: int


@dataclass(slots=True)
class DownloadsConfig:
    folder: str
    source_subdirectories: bool
    disc_subdirectories: bool
    concurrency: bool
    max_connections: int
    requests_per_minute: int
    verify_ssl: bool


@dataclass(slots=True)
class LastFmConfig:
    # The source on which to search for the tracks (use "deezer")
    source: str
    fallback_source: str


@dataclass(slots=True)
class CliConfig:
    text_output: bool
    progress_bars: bool
    max_search_results: int


@dataclass(slots=True)
class MiscConfig:
    version: str
    check_for_updates: bool


HOME = Path.home()
DEFAULT_DOWNLOADS_FOLDER = os.path.join(HOME, "StreamripDownloads")
DEFAULT_DOWNLOADS_DB_PATH = os.path.join(APP_DIR, "downloads.db")
DEFAULT_FAILED_DOWNLOADS_DB_PATH = os.path.join(APP_DIR, "failed_downloads.db")
BLANK_CONFIG_PATH = os.path.join(os.path.dirname(__file__), "config.toml")
assert os.path.isfile(BLANK_CONFIG_PATH), "Template config not found"


@dataclass(slots=True)
class ConfigData:
    toml: TOMLDocument
    downloads: DownloadsConfig

    deezer: DeezerConfig
    lastfm: LastFmConfig

    filepaths: FilepathsConfig
    artwork: ArtworkConfig
    metadata: MetadataConfig

    cli: CliConfig
    database: DatabaseConfig
    conversion: ConversionConfig

    misc: MiscConfig

    _modified: bool = False

    @classmethod
    def from_toml(cls, toml_str: str):
        toml = parse(toml_str)
        if (v := toml["misc"]["version"]) != CURRENT_CONFIG_VERSION:
            raise OutdatedConfigError(
                f"Need to update config from {v} to {CURRENT_CONFIG_VERSION}",
            )

        downloads = DownloadsConfig(**toml["downloads"])
        deezer = DeezerConfig(**toml["deezer"])
        lastfm = LastFmConfig(**toml["lastfm"])
        artwork = ArtworkConfig(**toml["artwork"])
        filepaths = FilepathsConfig(**toml["filepaths"])
        metadata = MetadataConfig(**toml["metadata"])
        cli = CliConfig(**toml["cli"])
        database = DatabaseConfig(**toml["database"])
        conversion = ConversionConfig(**toml["conversion"])
        misc = MiscConfig(**toml["misc"])

        return cls(
            toml=toml,
            downloads=downloads,
            deezer=deezer,
            lastfm=lastfm,
            artwork=artwork,
            filepaths=filepaths,
            metadata=metadata,
            cli=cli,
            database=database,
            conversion=conversion,
            misc=misc,
        )

    @classmethod
    def defaults(cls):
        with open(BLANK_CONFIG_PATH) as f:
            return cls.from_toml(f.read())

    def set_modified(self):
        self._modified = True

    @property
    def modified(self):
        return self._modified

    def update_toml(self):
        update_toml_section_from_config(self.toml["downloads"], self.downloads)
        update_toml_section_from_config(self.toml["deezer"], self.deezer)
        update_toml_section_from_config(self.toml["lastfm"], self.lastfm)
        update_toml_section_from_config(self.toml["artwork"], self.artwork)
        update_toml_section_from_config(self.toml["filepaths"], self.filepaths)
        update_toml_section_from_config(self.toml["metadata"], self.metadata)
        update_toml_section_from_config(self.toml["cli"], self.cli)
        update_toml_section_from_config(self.toml["database"], self.database)
        update_toml_section_from_config(self.toml["conversion"], self.conversion)

    def get_source(self, source: str) -> DeezerConfig:
        if source == "deezer":
            return self.deezer
        raise Exception(f"Invalid source {source}")


def update_toml_section_from_config(toml_section, config):
    for field in fields(config):
        toml_section[field.name] = getattr(config, field.name)


class Config:
    def __init__(self, path: str, /):
        self.path = path

        with open(path) as toml_file:
            self.file: ConfigData = ConfigData.from_toml(toml_file.read())

        self.session: ConfigData = copy.deepcopy(self.file)

    def save_file(self):
        if not self.file.modified:
            return

        with open(self.path, "w") as toml_file:
            self.file.update_toml()
            toml_file.write(dumps(self.file.toml))

    @staticmethod
    def _update_file(old_path: str, new_path: str):
        """Updates the current config based on a newer config `new_toml`."""
        with open(new_path) as new_conf:
            new_toml = parse(new_conf.read())

        toml_set_user_defaults(new_toml)

        with open(old_path) as old_conf:
            old_toml = parse(old_conf.read())

        update_config(old_toml, new_toml)

        with open(old_path, "w") as f:
            f.write(dumps(new_toml))

    @classmethod
    def update_file(cls, path: str):
        cls._update_file(path, BLANK_CONFIG_PATH)

    @classmethod
    def defaults(cls):
        return cls(BLANK_CONFIG_PATH)

    def __enter__(self):
        return self

    def __exit__(self, *_):
        self.save_file()


def set_user_defaults(path: str, /):
    """Update the TOML file at the path with user-specific default values."""
    shutil.copy(BLANK_CONFIG_PATH, path)

    with open(path) as f:
        toml = parse(f.read())

    toml_set_user_defaults(toml)

    with open(path, "w") as f:
        f.write(dumps(toml))


def toml_set_user_defaults(toml: TOMLDocument):
    toml["downloads"]["folder"] = DEFAULT_DOWNLOADS_FOLDER
    toml["database"]["downloads_path"] = DEFAULT_DOWNLOADS_DB_PATH
    toml["database"]["failed_downloads_path"] = DEFAULT_FAILED_DOWNLOADS_DB_PATH


def _get_dict_keys_r(d: dict) -> set[tuple]:
    keys = d.keys()
    ret = set()
    for cur in keys:
        val = d[cur]
        if isinstance(val, dict):
            ret.update((cur, *remaining) for remaining in _get_dict_keys_r(val))
        else:
            ret.add((cur,))
    return ret


def _nested_get(dictionary, *keys, default=None):
    return functools.reduce(
        lambda d, key: d.get(key, default) if isinstance(d, dict) else default,
        keys,
        dictionary,
    )


def _nested_set(dictionary, *keys, val):
    """Nested set. Throws exception if keys are invalid."""
    assert len(keys) > 0
    final = functools.reduce(lambda d, key: d.get(key), keys[:-1], dictionary)
    final[keys[-1]] = val


def update_config(old_with_data: dict, new_without_data: dict):
    """Used to update config when a new config version is detected."""
    old_keys = _get_dict_keys_r(old_with_data)
    new_keys = _get_dict_keys_r(new_without_data)
    common = old_keys.intersection(new_keys)
    common.discard(("misc", "version"))

    for k in common:
        old_val = _nested_get(old_with_data, *k)
        _nested_set(new_without_data, *k, val=old_val)
