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
import json

# MAX_FLOOD_WAIT = 300

CSV_FLUSHED = False

def message(csvf: TextIO|None, type: str, code: str, message: str, **details):
    '''
    this method calls flush on csv file every time if csvf is not None
    
    if you want to disable this behavior, pass None as csvf
    '''
    
    
    global CSV_FLUSHED
    obj: Dict[str, Any] = {}
    obj = {'code': code, 'message': message, 'details': details or None}
    ret: Dict[str, Dict[str, Any]] = {}
    out = sys.stdout
    if type[0] == 'e':
        ret = {'error': obj}
        out = sys.stderr
    elif type[0] == 'w':
        ret = {'warn': obj}
    elif type[0] == 'i':
        ret = {'info': obj}
    else:
        ret = {'log': obj}
    out.write(json.dumps(ret) + '\n')
    out.flush()
    if csvf is not None and not CSV_FLUSHED:
        try:
            csvf.flush()
            os.fsync(csvf.fileno())
        except OSError as e:
            sys.stdout.write(json.dumps({'warn': 
                {'code': 'CSV_FLUSH_ERROR', 
                'message': 'something went wrong when flushing',
                'details': None}}) + '\n')
            sys.stdout.flush()
        CSV_FLUSHED = True
    if type[0] == 'e':
        sys.exit(1)

async def flood_wait_or_exit(value: int, csvf: TextIO) -> None:
    message(csvf, 'warn', 'FLOOD_WAIT', 'got flood wait', value=value)
    
    # if value > MAX_FLOOD_WAIT:
    #     print(f'flood wait too long ({value} seconds), stopping', file=sys.stderr)
    #     flush_and_sync(f)
    #     sys.exit(1)
    
    try:
        await asyncio.sleep(value + 1)
    except asyncio.CancelledError:
        message(None, 'error', 'TASK_CANCELLED', 'task cancelled during wait, try again')
        
def exit_on_rpc(e: errors.RPCError, csvf: TextIO, msg: str) -> None:
    message(csvf,'error', 'RPC_ERROR', msg, c=e.CODE, m=e.MESSAGE, id=e.ID)