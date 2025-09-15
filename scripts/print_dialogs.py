import argparse
import asyncio
from typing import Optional, List, Tuple, Any, Dict
from config.config import get_tdlib_options
import os

from pyrogram import Client, errors, types, enums

SESSION = os.path.join(os.getcwd(), "test_account")

def parse_args() -> argparse.Namespace:
    p = argparse.ArgumentParser(description="Print dialogs to find chat ids")
    p.add_argument("--limit", type=int, default=50, help="number of dialogs to print, default 50")
    
    return p.parse_args()

async def main():
    args = parse_args()
    options: Dict[str, Any] = get_tdlib_options()
    api_id: int = options["api_id"]
    api_hash: str = options["api_hash"]    

    dialogs: List[types.Dialog] = []
    async with Client(SESSION, api_id, api_hash) as cl:
        async for d in cl.get_dialogs(limit=args.limit):
            dialogs.append(d)
    
    for i, d in enumerate(dialogs):
        chat: types.Chat = d.chat
        title = getattr(chat, 'title', None) or getattr(chat, 'first_name', None) or '(no title)'
        username = getattr(chat, 'username', None)
        print(f'[{i+1}] id={chat.id}, type={chat.type}, title={title}, username={("@"+username) if username else ""}')

if __name__ == '__main__':
    asyncio.run(main())