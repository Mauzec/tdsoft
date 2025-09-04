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


# TODO: need to do something with flood_wait (add, ex, retry button in ui )


def parse_args() -> argparse.Namespace:
    p = argparse.ArgumentParser(description="get public members of a TG chat")
    p.add_argument(
        # TODO: invite links not supported
        "chat", help="username, t.me/username, invite link(not supported yet)"
    )
    p.add_argument('--limit', type=int, default=1000, 
                   help='maximum number of members to retrieve; default is 50000; (MAX: 50000)')
    
    p.add_argument(
        "--output", type=str, 
        help="output csv file path; default is get-members-<time>.csv")
    
    p.add_argument(
        "--parse-from-messages", action='store_true',
        help="parse members from last --messages-limit messages")
    p.add_argument("--messages-limit", type=int, default=10,
                   help="number of messages to parse from, default 10; (MAX 5000)")

    # not supported 
    p.add_argument('--full-users', type=int, help='''
                   (EXPENSIVE OPERATION) expand user information 
                   (bio, blocked, premium, premium_gifts and etc);''')
    
    # TODO
    p.add_argument(
        '--exclude-bots', action='store_true', help='exclude bots from the output'
    )
    
    p.add_argument(
        '--parse-bio', action='store_true', 
        help='(+1 time) add user/bot bio to the output'
    )
    
    p.add_argument(
        '--add-additional-info', action='store_true',
        help= 'add user/bot additional info to the output (bio, premium, scam flag, etc)'
    )

    # not supported yet
    p.add_argument(
        '--auto-join', action='store_true', help='automatically join the chat if not a member')
    return p.parse_args()
    

def status_is_member(status: enums.ChatMemberStatus) -> bool:
    return status not in [
        enums.ChatMemberStatus.LEFT,
        enums.ChatMemberStatus.RESTRICTED,
        enums.ChatMemberStatus.BANNED
    ]

def write_row(writer: csv.DictWriter, u: types.User, 
              is_member: bool, bio: str|None=None,
              additional_info: List[str]|None=None) -> None:
    '''
    pass bio as None if not args.parse_bio,
    otherwise as str('' if bio not available)
    
    pass additional_info as None if not args.add_additional_info,
    otherwise as List[str] (if item is not available, pass '' for it)
    '''
    
    row = [u.id, u.username or '', u.first_name or '', u.last_name or '', 
           'member' if is_member else 'not member',
           'bot' if u.is_bot else 'user']
    if bio is not None:
        row.append(bio)
    if additional_info is not None:
        row.extend(additional_info)
    
    writer.writerow(row)
    
def get_additional_info(u: types.User) -> List[str]:
    info = []
    info.append('premium' if u.is_premium else 'not premium')
    info.append('is_deleted' if u.is_deleted else 'not deleted')
    info.append('scam' if u.is_scam else 'not scam')
    info.append('verified' if u.is_verified else 'not verified')
    info.append(u.phone_number or '')
    return info
    
async def fetch_bio(u: types.User, app: Client) -> str:
    try:
        chat: types.Chat = await app.get_chat(u.id)
        return chat.bio or ''
    except errors.RPCError as e:
        exit_on_rpc(e, sys.stderr)
    return ''
        

async def fetch_members(
    app: Client,
    name: str,
    args: argparse.Namespace,
) -> Dict[int, types.User]:
    '''
    Returns a set of user ids
    '''
    
    users: Dict[int, types.User] = {}
    
    chat = await app.get_chat(name)
    chat_type = chat.type
    
    total = 0
    with open(args.output, 'a', newline='', encoding='utf-8') as f:
        writer = csv.writer(f)
        
        async def _write_members() -> None:
            nonlocal total
            try:
                async for m in app.get_chat_members(chat.id, limit=args.limit):
                    u: types.User = m.user
                    if int(u.id) in users:
                        continue
                    is_member = status_is_member(m.status)
                    
                    bio: str|None = None
                    if args.parse_bio:
                        if not u.is_bot:
                            bio = await fetch_bio(u, app)
                        else:
                            bio = ''

                    additional_info: List[str]|None = None
                    if args.add_additional_info:
                        additional_info = get_additional_info(u)
                    
                    write_row(writer, u, is_member, bio, additional_info)                    
                    
                    total += 1
                    users[int(u.id)] = u
                return None
            
            except errors.FloodWait as e:
                value = int(getattr(e, 'value', 0))
                await flood_wait_or_exit(value, f, f'parsed {total} members so far, saved to {args.output}')
                  
            except errors.RPCError as e:
                exit_on_rpc(e, f)
        await _write_members()
        print(f'parsed {total} members, saved to {args.output}', file=sys.stderr)

    return users
        
    
async def fetch_members_from_messages(
    app: Client,
    name: str,
    args: argparse.Namespace,
    
    users: Dict[int, types.User],
) -> None:
    with open(args.output, 'a',newline='', encoding='utf-8') as f:
        writer = csv.writer(f)
        
        # FIXME: change to 100 later
        page_size = 50
        
        offset_id = 0
        
        total_messages = 0
        async def _write_members_from_messages() -> None:
            nonlocal total_messages
            nonlocal offset_id
            nonlocal page_size
            
            while total_messages < args.messages_limit:
                curr_messages = 0
                last_msg_id = None
                if args.messages_limit - total_messages < page_size:
                    page_size = args.messages_limit - total_messages
                try:
                    history: AsyncGenerator[types.Message, None] = \
                        app.get_chat_history(name, limit=page_size, offset_id=offset_id)
                    async for m in history:
                        total_messages += 1
                        last_msg_id = m.id
                        curr_messages += 1
                        
           
                        if not m.from_user:
                            continue
                        
                        u: types.User = m.from_user
                        
                        if int(u.id) in users:
                            continue
                        
                        users[int(u.id)] = u
                        
                        # check membership status
                        is_member = False
                        try:
                            member: types.ChatMember = await app.get_chat_member(name, u.id)
                            is_member = status_is_member(member.status)
                        except errors.RPCError as e:
                            exit_on_rpc(e, f)
                        except errors.FloodWait as e:
                            value = int(getattr(e, 'value', 0))
                            await flood_wait_or_exit(value, f, f'Parsed {total_messages-1} messages so far, saved to {args.output}')
                        
                        bio: str|None = None
                        if args.parse_bio:
                            if not u.is_bot:
                                bio = await fetch_bio(u, app)
                            else:
                                bio = ''
                        additional_info: List[str]|None = None
                        if args.add_additional_info:
                            additional_info = get_additional_info(u)
                        
                        write_row(writer, u, is_member, bio, additional_info)
                        
                        if total_messages >= args.messages_limit:
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
                
        await _write_members_from_messages()
        
        print(f'parsed {total_messages} messages, saved to {args.output}', file=sys.stderr)
        print(f'total unique users found: {len(users)}', file=sys.stderr)
    return None
    

async def main():
    args = parse_args()
    if args.limit > 50000:
        print(f'limit too high: {args.limit}, maximum is 50000', file=sys.stderr)
        sys.exit(1)
    
    if args.parse_from_messages and args.messages_limit > 10000:
        print(f'messages limit too high: {args.messages_limit}, maximum is 10000', file=sys.stderr)
        sys.exit(1)    
    
    if not args.output:
        args.output = f'get-members-{int(time.time())}.csv'
    
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
        
    columns = ['user_id', 'username', 'first_name', 'last_name', 'is_member', 'is_bot']
    if args.parse_bio:
        columns.append('bio')
    if args.add_additional_info:
        columns.extend(['premium_status', 'is_deleted', 'is_scam', 'is_verified', 'phone_number'])
    with open(args.output, 'a', newline='', encoding='utf-8') as f:
        writer = csv.writer(f)
        writer.writerow(columns)

    async with Client('my_account', api_id, api_hash) as app:
        users: Dict[int, types.User] = await fetch_members(app, name, args)
    
        if args.parse_from_messages:
            await fetch_members_from_messages(app, name, args, users)
    
        
        
if __name__ == '__main__':
    asyncio.run(main())