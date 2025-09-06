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
from utils.io import flood_wait_or_exit, exit_on_rpc, flush_and_sync
from datetime import datetime, date
from collections import defaultdict
from statistics import median


def parse_args() -> argparse.Namespace:
    p = argparse.ArgumentParser(
        description='''Get chat statistics. 
        WARNING: without --limit-history it will read all history of the chat. '''
    )
    p.add_argument(
        # TODO: invite links not supported
        "chat", help="username, t.me/username, invite link(not supported yet)"
    )
    p.add_argument(
        "--output", type=str, default=f'get-statistics-{int(time.time())}.csv',
        help="output csv file path; default is get-chat-statistics-<time>.csv")
    
    p.add_argument(
        '--limit-history', type=int, default=0,
        help='limit number of messages to parse from history; default is 0 (all history)'
    )
    
    return p.parse_args()


COLUMNS=['title', 'username', 'public_members_count', 'bio', 'is_verified',
         'is_fake', 'is_scam', 'can_forward', 'invite_link',
         'total_messages', 'day_median', 'week_median', 'weekday_median']


# TODO: add something more
class HistoryStatistics:
    def __init__(self) -> None:
        self.total_messages: int = 0
        self.weekday_median: float = 0.0
        self.week_median: float = 0.0
        self.day_median: float = 0.0
        self.top5_msg_senders: List[Tuple[int, int]] = [] # (user_id, count)
    
        
async def fetch_history_statistics(
    app: Client,
    chat_id: str,
    args: argparse.Namespace
) -> HistoryStatistics:
    with open(args.output, 'a', newline='', encoding='utf-8') as f:
        writer = csv.writer(f)
    
        total_messages = 0
        page_size = 50
        offset_id = 0
        
        msg_per_day: defaultdict = defaultdict(int) # date: int
        msg_per_week: defaultdict = defaultdict(int) # (int,int): int
        msg_per_weekday: defaultdict = defaultdict(int) # int: int
        
        top_msg_senders: defaultdict = defaultdict(int) # str: int
        
        while (args.limit_history > 0 and total_messages < args.limit_history) or True:
            curr_messages = 0
            last_msg_id: Optional[int] = None
            if args.limit_history - total_messages < page_size:
                page_size = args.limit_history - total_messages
                
            try: 
                history: AsyncGenerator[types.Message, None] = \
                    app.get_chat_history(chat_id, page_size, offset_id=offset_id)
                
                async for msg in history:
                    
                    d: date = msg.date.date()
                    
                    msg_per_day[d] += 1
                    y, w, _ = d.isocalendar()
                    msg_per_week[(y, w)] += 1
                    weekday = d.weekday()
                    msg_per_weekday[weekday] += 1
                    
                    if msg.from_user:
                        top_msg_senders[msg.from_user.username] += 1

                    total_messages += 1
                    last_msg_id = msg.id
                    curr_messages += 1
                    
                    if (args.limit_history > 0 and total_messages < args.limit_history):
                        break
            
            except errors.RPCError as e:
                exit_on_rpc(e, f)
            except errors.FloodWait as e:
                value = int(getattr(e, 'value', 0))
                await flood_wait_or_exit(value, f, f'parsed {total_messages} messages so far, saved to {args.output}')
            
            if curr_messages == 0 or last_msg_id is None or last_msg_id <= 1:
                print(f'no more messages to parse, stopping', file=sys.stderr)
                break
            offset_id = last_msg_id   

        stats = HistoryStatistics()
        stats.total_messages = total_messages
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
            ]
            for user_id, count in history_stats.top5_msg_senders:
                print(f'top sender: {user_id} with {count} messages')
            
            writer.writerow(row)
            flush_and_sync(f)
            
        except errors.FloodWait as e:
            flood_wait_or_exit(int(getattr(e, 'value', 0)), f,
                               f'during get_statistics')
            
        except errors.RPCError as e:
            exit_on_rpc(e, f)


async def main():
    args = parse_args()

    options: Dict[str, Any] = get_tdlib_options()
    api_id: int = options["api_id"]
    api_hash: str = options["api_hash"]    
    
    kind, name = parse_chat_name(args.chat)
    if kind == ChatNameKind.EMPTY:
        print(f'invalid chat name: {args.chat}', file=sys.stderr)
        sys.exit(1)
    if kind == ChatNameKind.INVITE_LINK:
        print(f'invite links are not supported yet: {args.chat}', file=sys.stderr)
        sys.exit(1)
        
    with open(args.output, 'w', newline='', encoding='utf-8') as f:
        writer = csv.writer(f)
        writer.writerow(COLUMNS)
        
    async with Client('my_account', api_id, api_hash) as app:
        await get_statistics(app, name, args)
        
    print(f'saved statistics to {args.output}', file=sys.stderr)
        
if __name__ == '__main__':
    asyncio.run(main())