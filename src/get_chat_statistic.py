import sys
import csv
import argparse
import asyncio
import time
from typing import Optional, List, Tuple, Any, Dict, Set
from config.config import get_tdlib_options
from utils.chatname import parse_chat_name, ChatNameKind
from pyrogram import Client, errors, types, enums
from typing import AsyncGenerator, TextIO
from utils.io import flood_wait_or_exit, exit_on_rpc


def parse_args() -> argparse.Namespace:
    p = argparse.ArgumentParser(
        description='Get chat statistics'
    )


async def main():
    args = parse_args()


if __name__ == '__main__':
    asyncio.run(main())