import os
import sys
import csv
import argparse
import asyncio
from typing import Optional, List, Tuple, Any, Dict
from config.config import get_tdlib_options
from pyrogram import Client, errors, types, enums
import regex as re
from enum import Enum, auto
from urllib.parse import urlparse, ParseResult
from typing import TextIO


MAX_FLOOD_WAIT = 300

def flush_and_sync(f: TextIO) -> None:
    try:
        f.flush()
        os.fsync(f.fileno())
    except OSError as e:
        print(f'detected error during fsync: {e}', file=sys.stderr)

async def flood_wait_or_exit(value: int, f: TextIO, progress_msg: str='') -> None:
    print(f'got flood_wait: {value} seconds', file=sys.stderr)
    if progress_msg:
        print(progress_msg, file=sys.stderr)
    
    if value > MAX_FLOOD_WAIT:
        print(f'flood wait too long ({value} seconds), stopping', file=sys.stderr)
        flush_and_sync(f)
        sys.exit(1)
    
    print(f'waiting {value} seconds...', file=sys.stderr)
    print(f'WARNING: task may be cancelled by system', file=sys.stderr)
    
    try:
        await asyncio.sleep(value + 1)
    except asyncio.CancelledError:
        print(f'task cancelled during wait', file=sys.stderr)
        flush_and_sync(f)
        sys.exit(1)

def exit_on_rpc(e: Exception, f) -> None:
    print(f'got rpc error: {e}', file=sys.stderr)
    flush_and_sync(f)
    sys.exit(1)