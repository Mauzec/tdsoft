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

SESSION = os.path.join(os.getcwd(), "test_account")

def parse_args() -> argparse.Namespace:
    p = argparse.ArgumentParser(
        description='''Get chat statistics. 
        WARNING: without --history-limit it will read all history of the chat. ''')
    p.add_argument(
        # TODO: invite links not supported
        "chat", help="username, t.me/username, invite link(not supported yet), id")
    p.add_argument(
        "--output", type=str, default=f'get-statistics-{int(time.time())}.csv',
        help="output csv file path; default is get-chat-statistics-<time>.csv")
    
    p.add_argument(
        '--history-limit', type=int, default=0,
        help='limit number of messages to parse from history; default is 0 (all history)')
    
    return p.parse_args()


COLUMNS=['title', 'username', 'public_members_count', 'bio', 'is_verified',
         'is_fake', 'is_scam', 'can_forward', 'invite_link',
         'total_messages', 'day_median', 'week_median', 'weekday_median',
         '', '', '', 'top5', 'msg_senders']


# TODO: add something more
class HistoryStatistics:
    def __init__(self) -> None:
        self.total_messages: int = 0
        self.weekday_median: float = 0.0
        self.week_median: float = 0.0
        self.day_median: float = 0.0
        self.top5_msg_senders: List[Tuple[str, int]] = [] # (username, count)
    
        
async def fetch_history_statistics(
    app: Client,
    chat_id: str,
    args: argparse.Namespace
) -> HistoryStatistics:
    total_messages = 0
    actual_messages = 0
    page_size = 50
    offset_id = 0
    
    msg_per_day: defaultdict[date, int] = defaultdict(int)
    msg_per_week: defaultdict[int, int] = defaultdict(int)
    msg_per_weekday: defaultdict[int, int] = defaultdict(int)
    
    top_msg_senders: defaultdict[str, int] = defaultdict(int)
    while (total_messages < args.history_limit) or (args.history_limit <= 0):

        curr_messages = 0
        last_msg_id: Optional[int] = None
        if args.history_limit > 0 and args.history_limit - total_messages < page_size:
            page_size = args.history_limit - total_messages
            
        try: 
            history: AsyncGenerator[types.Message, None] = \
                app.get_chat_history(chat_id, page_size, offset_id=offset_id)
            
            async for msg in history:
    
                if not msg.service and not msg.empty:
                    d: date = msg.date.date()
                    
                    msg_per_day[d] += 1
                    y, w, _ = d.isocalendar()
                    msg_per_week[(y, w)] += 1
                    weekday = d.weekday()
                    msg_per_weekday[weekday] += 1
                    
                    if (msg.text or msg.media):
                        if msg.media:
                            io.message(None, 'info', 'MEDIA_MESSAGE',
                                        when='fetching history statistics',
                                        media_type=msg.media.value, msg_id=msg.id)
                        username: str = 'UNKNOWN'
                        if msg.from_user:
                            if msg.from_user.username:
                                username = msg.from_user.username or 'UNKNOWN'
                            elif msg.from_user.first_name or msg.from_user.last_name:
                                username = (msg.from_user.first_name or '') + ' ' + (msg.from_user.last_name or '')
                        elif msg.sender_chat.username:
                            username = msg.sender_chat.username
                        elif msg.sender_chat.title:
                            username = msg.sender_chat.title
                        top_msg_senders[username] += 1
                        
                    actual_messages += 1
                
                total_messages += 1
                last_msg_id = msg.id
                curr_messages += 1
                
                if (args.history_limit > 0 and total_messages >= args.history_limit):
                    break
        
        except errors.FloodWait as e:
            await io.flood_wait_or_exit(None, int(getattr(e, 'value', 0)), 'fetching history statistics')
            
        except errors.RPCError as e:
            io.exit_on_rpc(None, e, 'fetching history statistics')
            
        except Exception as e:
            io.message(None, 'error', 'UNEXPECTED_ERROR',
                        when='fetching history statistics',
                        error=str(e))
        
        if curr_messages == 0 or last_msg_id is None or last_msg_id <= 1:
            break
        offset_id = last_msg_id   

    stats = HistoryStatistics()
    stats.total_messages = actual_messages
    stats.day_median = median(msg_per_day.values())
    stats.week_median = median(msg_per_week.values())
    stats.weekday_median = median(msg_per_weekday.values())
    stats.top5_msg_senders = \
        sorted(top_msg_senders.items(), key=lambda x: x[1], 
                reverse=True)[:min(5, len(top_msg_senders))]
    return stats
        
    
async def get_statistics(
    app: Client,
    chat_id: str,
    args: argparse.Namespace
) -> None:
    with open(args.output, 'a', newline='', encoding='utf-8') as f:
        writer = csv.writer(f)
        try:
            chat = await app.get_chat(chat_id)
            row = [
                chat.title or 'UNKNOWN', chat.username or 'UNKNOWN', chat.members_count or 'UNKNOWN',
                chat.bio or 'no bio', 'yes' if chat.is_verified else 'no',
                'yes' if chat.is_fake else 'no', 'yes' if chat.is_scam else 'no', 
                'yes' if chat.has_protected_content else 'no',
                chat.invite_link or 'UNKNOWN']

            history_stats = await fetch_history_statistics(app, chat_id, args)
            row += [
                history_stats.total_messages,
                history_stats.day_median,
                history_stats.week_median,
                history_stats.weekday_median,
                '', '', '', 'username', 'count'
            ]
            
            writer.writerow(row)
            for u, c in history_stats.top5_msg_senders:
                writer.writerow(['']*(len(COLUMNS)-2) + [u or 'UNKNOWN', c])
            io.CSV_FLUSHED = False
                        
        except errors.RPCError as e:
            io.exit_on_rpc(f, e, 'getting statistics')
            
        except Exception as e:
            io.message(f, 'error', 'UNEXPECTED_ERROR',
                        when='getting statistics',
                        error=str(e))
            
    io.CSV_FLUSHED = True


async def main():
    io.message(None, 'info', 'SCRIPT_STARTED', script='get_members.py')
    args = parse_args()

    options: Dict[str, Any] = get_tdlib_options()
    api_id: int = options["api_id"]
    api_hash: str = options["api_hash"]    
    
    kind, name = parse_chat_name(args.chat)
    if kind == ChatNameKind.EMPTY:
        io.message(None, 'error', 'INVALID_CHAT_NAME', name=args.chat)
        
    if kind == ChatNameKind.INVITE_LINK:
        io.message(None, 'error', 'INVITE_LINK_NOT_SUPPORTED', name=args.chat)
        
    with open(args.output, 'w', newline='', encoding='utf-8') as f:
        writer = csv.writer(f)
        writer.writerow(COLUMNS)
    io.CSV_FLUSHED = True
        
    async with Client(SESSION, api_id, api_hash) as app:
        await get_statistics(app, name, args)
        
        io.message(None, 'info', 'ALL_DONE', output=os.path.abspath(args.output))
        
    
        
if __name__ == '__main__':
    try:
        asyncio.run(main())
    except Exception as e:
        io.message(None, 'error', 'UNEXPECTED_ERROR', 
                    when='main', error=str(e))