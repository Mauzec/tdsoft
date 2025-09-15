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

CSV_FLUSHED = False

def message(csvf: TextIO|None, msg_type: str, code: str, **details):
    '''
    !!! this method calls flush on csv file every time if csvf is not None
    
    if you want to disable this behavior, pass None as csvf
    '''
    
    
    global CSV_FLUSHED
    obj: Dict[str, Any] = {}
    obj = {'code': code, 'details': details or None}
    ret: Dict[str, Dict[str, Any]] = {}
    out = sys.stdout
    if msg_type[0] == 'e':
        ret = {'error': obj}
        out = sys.stderr
    elif msg_type[0] == 'w':
        ret = {'warn': obj}
    elif msg_type[0] == 'i':
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
                'details': None}}) + '\n')
            sys.stdout.flush()
        CSV_FLUSHED = True
    if msg_type[0] == 'e':
        sys.exit(1)

async def flood_wait_or_exit(csvf: TextIO, value: int, when:str) -> None:
    message(csvf, 'warn', 'FLOOD_WAIT', when=when, value=value)
    
    # if value > MAX_FLOOD_WAIT:
    #     print(f'flood wait too long ({value} seconds), stopping', file=sys.stderr)
    #     flush_and_sync(f)
    #     sys.exit(1)
    
    try:
        await asyncio.sleep(value + 1)
    except asyncio.CancelledError:
        message(None, 'error', 'TASK_CANCELLED', when='flood_wait_or_exit')
        
def exit_on_rpc(csvf: TextIO, e: errors.RPCError, when: str) -> None:
    message(csvf,'error', 'RPC_ERROR', when=when, c=e.CODE, m=e.MESSAGE, id=e.ID)