import logging
from abc import ABC, abstractmethod

from rich.prompt import Prompt

from ..client import Client, DeezerClient
from ..config import Config
from ..console import console
from ..exceptions import AuthenticationError

logger = logging.getLogger("streamrip")


class CredentialPrompter(ABC):
    client: Client

    def __init__(self, config: Config, client: Client):
        self.config = config
        self.client = self.type_check_client(client)

    @abstractmethod
    def has_creds(self) -> bool:
        raise NotImplementedError

    @abstractmethod
    async def prompt_and_login(self):
        """Prompt for credentials and log into the client."""
        raise NotImplementedError

    @abstractmethod
    def save(self):
        """Save current credentials to the config file."""
        raise NotImplementedError

    @abstractmethod
    def type_check_client(self, client: Client):
        raise NotImplementedError


class DeezerPrompter(CredentialPrompter):
    client: DeezerClient

    def has_creds(self) -> bool:
        return self.config.session.deezer.arl != ""

    async def prompt_and_login(self):
        if not self.has_creds():
            self._prompt_and_set_arl()
        while True:
            try:
                await self.client.login()
                break
            except AuthenticationError:
                console.print("[yellow]Invalid ARL, try again.")
                self._prompt_and_set_arl()
        self.save()

    def _prompt_and_set_arl(self):
        console.print(
            "If you're not sure how to find the ARL cookie, see the instructions at ",
            "[blue underline]https://github.com/nathom/streamrip/wiki/Finding-your-Deezer-ARL-Cookie",
        )
        self.config.session.deezer.arl = Prompt.ask("Enter your [bold]ARL")

    def save(self):
        c = self.config.session.deezer
        cf = self.config.file.deezer
        cf.arl = c.arl
        self.config.file.set_modified()
        console.print(
            f"[green]Credentials saved to config file at [bold cyan]{self.config.path}",
        )

    def type_check_client(self, client) -> DeezerClient:
        assert isinstance(client, DeezerClient)
        return client


PROMPTERS = {
    "deezer": DeezerPrompter,
}


def get_prompter(client: Client, config: Config) -> CredentialPrompter:
    """Return a CredentialPrompter for the given client."""
    p = PROMPTERS.get(client.source)
    if p is None:
        raise Exception(f"No prompter for source '{client.source}'")
    return p(config, client)
