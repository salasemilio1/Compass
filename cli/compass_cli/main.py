import click

from compass_cli.commands.init import init


@click.group()
def cli():
    """compass -- in-repo agent-memory protocol."""


cli.add_command(init)
