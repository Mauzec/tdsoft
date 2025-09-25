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
from datetime import datetime, date
from collections import defaultdict
from statistics import median


'''
Search messages in a group/channel/private
'''

COLUMNS = ['message_id', 'text', 'date']

def parse_args() -> argparse.Namespace:
    p = argparse.ArgumentParser(
        description='''Search messages in a group/channel/private chat by keywords. 
        WARNING: without --limit-history it will read all history of the chat. ''')
    
    p.add_argument( # for future use
        'session', type=str, help='session path (string)'
    )
    p.add_argument(
        "chat", help="username, t.me/username, id")
    p.add_argument(
        'username', help='username of the user')

    p.add_argument(
        "--output", type=str,
        help="output csv file path; default is search-messages-username-<time>.csv")
    
    p.add_argument(
        '--from-date', type=str, help='date from which to start searching messages, format MM/DD/YYYY')
    p.add_argument(
        '--to-date', type=str, help='date to which to search messages, format MM/DD/YYYY')
    
    # TODO: implement it
    p.add_argument(
        '--keywords', type=str, nargs='+', help='keywords to search for in messages')
    
    return p.parse_args()
    
  
def get_media_content(msg: types.Message) -> str:
    if not msg.media:
        return ''
    
    if msg.media == enums.MessageMediaType.PHOTO:
        return 'photo'
    elif msg.media == enums.MessageMediaType.VIDEO:
        return 'video'
    elif msg.media == enums.MessageMediaType.VOICE:
        return 'voice'
    elif msg.media == enums.MessageMediaType.AUDIO:
        return 'audio'
    elif msg.media == enums.MessageMediaType.DOCUMENT:
        return 'document'
    elif msg.media == enums.MessageMediaType.STICKER:
        return 'sticker'
    elif msg.media == enums.MessageMediaType.ANIMATION:
        return 'animation'
    elif msg.media == enums.MessageMediaType.VIDEO_NOTE:
        return 'video_note'
    elif msg.media == enums.MessageMediaType.WEB_PAGE:
        return 'web_page'
    elif msg.media == enums.MessageMediaType.CONTACT:
        return 'contact'
    elif msg.media == enums.MessageMediaType.LOCATION:
        return 'location'
    elif msg.media == enums.MessageMediaType.POLL:
        return 'poll'
    elif msg.media == enums.MessageMediaType.GAME:
        return 'game'
    else:
        return 'unknown'
    
async def fetch_messages(
    app: Client,
    args: argparse.Namespace
) -> None:
    with open(args.output, 'a', newline='', encoding='utf-8') as f:
        writer = csv.writer(f)
        
        user_messages: int = 0
        page_size: int = 75
        offset_id: int = 0

        to_date = datetime.strptime(args.to_date, '%m/%d/%Y')
        from_date = datetime.strptime(args.from_date, '%m/%d/%Y')
        
        last_offset_date: datetime = to_date
        while last_offset_date >= from_date:
            last_msg_id: Optional[int] = None
            curr_messages: int = 0
            try:
                history: AsyncGenerator[types.Message, None] = \
                    app.get_chat_history(args.chat, limit=page_size, offset_date=last_offset_date, offset_id=offset_id)
                async for m in history:
                    if m.date < from_date:
                        last_offset_date = m.date
                        break
                    
                    if not m.service and not m.empty and (m.text or m.media):
                        username: str = ''
                        
                        if m.from_user and m.from_user.username:
                            username = m.from_user.username
                        elif m.sender_chat.username:
                            username = m.sender_chat.username
                        # io.message(None, 'debug', 'CHECKING_MESSAGE',
                        #             when='fetching messages',
                        #             msg_id=m.id,
                        #             date=m.date.strftime("%Y-%m-%d %H:%M:%S"),
                        #             username=username,)
                        if username == args.username:
                            media = get_media_content(m)
                            text = m.text or media
                            writer.writerow([m.id, text, m.date.strftime("%m.%d.%Y %H:%M:%S")])
                            io.CSV_FLUSHED = False
                            user_messages += 1
                        
                        
                    last_msg_id = m.id
                    last_offset_date = m.date
                    curr_messages += 1
                      
            except errors.FloodWait as e:
                await io.flood_wait_or_exit(f, int(getattr(e, 'value', 0)), 'fetching messages')
                
            except errors.RPCError as e:
                io.exit_on_rpc(f, e, 'fetching messages')
                
            except Exception as e:
                io.message(f, 'error', 'UNEXPECTED_ERROR',
                            when='fetching messages',
                            error=str(e))
                
            if curr_messages == 0 or last_msg_id is None or last_msg_id <= 1:
                break
            offset_id = last_msg_id
    
    io.message(None, 'info', 'MESSAGES_FETCHED', total=user_messages)  
    io.CSV_FLUSHED = True                        
    

async def main():
    io.message(None, 'info', 'SCRIPT_STARTED', script='search_messages.py')
    
    try:
        args = parse_args()
    except Exception as e:
        io.message(None, 'error', "ARGPARSE_ERROR", error=str(e))
    
    if not args.session:
        io.message(None, 'error', 'NO_SESSION', when='main')
        
    if not args.from_date:
        io.message(None, 'error', "FROM_DATE_REQUIRED", error="from_date is required")
    else:
        try:
            datetime.strptime(args.from_date, '%m/%d/%Y')
        except ValueError:
            io.message(None, 'error', "FROM_DATE_INVALID", error="from_date format is invalid, should be MM/DD/YYYY")
    if not args.to_date:
        io.message(None, 'error', "TO_DATE_REQUIRED", error="to_date is required")
    else:
        try:
            datetime.strptime(args.to_date, '%m/%d/%Y')
        except ValueError:
            io.message(None, 'error', "TO_DATE_INVALID", error="to_date format is invalid, should be MM/DD/YYYY")
            
    if not args.output:
        args.output = f'search-messages-{args.username}-{int(time.time())}.csv'
        
    options: Dict[str, Any] = get_tdlib_options()
    api_id: int = options["api_id"]
    api_hash: str = options["api_hash"]    
    
    kind, name = parse_chat_name(args.chat)
    if kind == ChatNameKind.EMPTY:
        io.message(None, 'error', 'INVALID_CHAT_NAME', name=args.chat)
    if kind == ChatNameKind.INVITE_LINK:
        io.message(None, 'error', 'INVITE_LINK_NOT_SUPPORTED', name=args.chat)
    args.chat = name
        
    kind, name = parse_chat_name(args.username)
    if kind == ChatNameKind.EMPTY or kind == ChatNameKind.CHAT_ID or kind == ChatNameKind.INVITE_LINK:
        io.message(None, 'error', 'INVALID_USERNAME', name=args.username)
    args.username = name
        
    with open(args.output, 'w', newline='', encoding='utf-8') as f:
        writer = csv.writer(f)
        writer.writerow(COLUMNS)
    io.CSV_FLUSHED = True
    
    async with Client(args.session, api_id=api_id, api_hash=api_hash) as app:
        await fetch_messages(app, args)
        
        io.message(None, 'info', 'ALL_DONE', output=os.path.abspath(args.output))


if __name__ == '__main__':
    try:
        asyncio.run(main())
    except Exception as e:
        io.message(None, 'error', 'UNEXPECTED_ERROR', 
                    when='main', error=str(e))
    
    