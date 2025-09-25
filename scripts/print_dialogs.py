import sys
import csv
import argparse
import asyncio
import time
import os
from typing import Optional, List, Tuple, Any, Dict, Set
from config.config import get_tdlib_options
from utils.chatname import parse_chat_name, ChatNameKind
from pyrogram import Client, errors, types, enums
from typing import AsyncGenerator, TextIO
import utils.io as io
from pyrogram import Client, errors, types, enums


def parse_args() -> argparse.Namespace:
    p = argparse.ArgumentParser(description="Print dialogs to find chat ids")
    p.add_argument( # for future use
        'session', type=str, help='session path (string)'
    )
    
    p.add_argument("--limit", type=int, default=50, help="number of dialogs to print, default 50")
    
    p.add_argument("--output", type=str, default="", 
                   help="path to output CSV file, default ./print-dialogs-<timestamp>.csv")
    
    return p.parse_args()

async def main():
    io.CSV_FLUSHED = True
    io.message(None, 'info', 'SCRIPT_STARTED', script='get_members.py')
    try:
        args = parse_args()
    except Exception as e:
        io.message(None, 'error', "ARGPARSE_ERROR", error=str(e))
    
    if not args.session:
        io.message(None, 'error', 'NO_SESSION', when='main')
        
    if not args.output:
        args.output = f'./print-dialogs-{int(time.time())}.csv'
        
    
    options: Dict[str, Any] = get_tdlib_options()
    api_id: int = options["api_id"]
    api_hash: str = options["api_hash"]    

    dialogs: List[types.Dialog] = []
    with open(args.output, 'w', newline='', encoding='utf-8') as f:
        writer = csv.writer(f)
        writer.writerow(['chat_id', 'type', 'title', 'username'])
        
        async with Client(args.session, api_id, api_hash) as cl:
            try:
                async for d in cl.get_dialogs(limit=args.limit):
                    dialogs.append(d)
                    
            except errors.RPCError as e:
                io.exit_on_rpc(f, e, 'get dialogs')
            except Exception as e:
                io.message(f, 'error', 'UNEXPECTED_ERROR',
                            when='get dialogs',
                            error=str(e))
    
        for i, d in enumerate(dialogs):
            chat: types.Chat = d.chat
            title = getattr(chat, 'title', None) or getattr(chat, 'first_name', None) or '(no title)'
            username = getattr(chat, 'username', None)
            print(f'[{i+1}] id={chat.id}, type={chat.type}, title={title}, username={("@"+username) if username else ""}')
    io.CSV_FLUSHED = True
    
    io.message(None, 'info', 'ALL_DONE', output=os.path.abspath(args.output))

if __name__ == '__main__':
    try:
        asyncio.run(main())
    except Exception as e:
        io.message(None, 'error', 'UNEXPECTED_ERROR', 
                    when='main', error=str(e))