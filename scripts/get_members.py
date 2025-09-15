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

SESSION = os.path.join(os.getcwd(), "test_account")

# TODO: need to do something with flood_wait (add, ex, retry button in ui )
# TODO: add more errors, like chat not found, instead of just rpc error
# TODO: add check if messsage is service or not, or it's from group/channel or user

def parse_args() -> argparse.Namespace:
    p = argparse.ArgumentParser(description="get public members of a TG chat")
    p.add_argument(
        # TODO: invite links not supported
        "chat", help="username, t.me/username, invite link(not supported yet)"
    )
    p.add_argument('--limit', type=int, default=1000, 
                   help='maximum number of members to return (default is 1000, max is 50000)')
    
    p.add_argument(
        "--output", type=str, 
        help="output csv file path; default is get-members-<timestamp>.csv")
    
    p.add_argument(
        "--parse-from-messages", action='store_true',
        help="parse members from last --messages-limit messages")
    p.add_argument("--messages-limit", type=int, default=10,
                   help="number of messages to parse from, default 10; (MAX 5000)")

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
        '--auto-join', action='store_true', help='automatically join the chat if not a member',
    )
    
    # TODO: implement it, this useful for large groups and when no need parse all
    # p.add_argument(
    #     '--members-filter',
    # )
    
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
    
async def fetch_bio(csvf: TextIO, u: types.User, app: Client) -> str:
    chat: types.Chat|None = None
    while chat is None:
        try:
            chat = await app.get_chat(u.id)
        except errors.FloodWait as e:
            await io.flood_wait_or_exit(csvf, int(getattr(e, 'value', 0)), 'fetching bio')
            chat = None
        except errors.RPCError as e:
            io.exit_on_rpc(csvf, e, 'fetching bio')
        except Exception as e:
            io.message(csvf, 'error', 'UNEXPECTED_ERROR',
                       when='fetching bio', error=str(e))
    
    return chat.bio or ''

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
                            bio = await fetch_bio(f, u, app)
                        else:
                            bio = ''

                    additional_info: List[str]|None = None
                    if args.add_additional_info:
                        additional_info = get_additional_info(u)
                    
                    write_row(writer, u, is_member, bio, additional_info)
                    io.CSV_FLUSHED = False                 
                    
                    total += 1
                    users[int(u.id)] = u
            
            # if we get flood wait, we can't wait, because 
            # there is no offset parameters in get_chat_members
            # except errors.FloodWait as e:
            #     value = int(getattr(e, 'value', 0))
            #     await io.flood_wait_or_exit(f, value, 'fetching members')
            except errors.RPCError as e:
                io.exit_on_rpc(f, e, 'fetching members')
            except Exception as e:
                io.message(f, 'error', 'UNEXPECTED_ERROR',
                           when='fetching members', error=str(e))
        
        await _write_members()
        io.message(None, 'info', 'MEMBERS_FETCHED', total=total)
        
    io.CSV_FLUSHED = True
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
        page_size = 100
        
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
                        if not m.from_user:
                            continue

                        u: types.User = m.from_user

                        if int(u.id) in users:
                            continue
                        
                        member: types.ChatMember = await app.get_chat_member(name, u.id)
                        is_member = status_is_member(member.status)
                        
                        bio: str|None = None
                        if args.parse_bio:
                            if not u.is_bot:
                                bio = await fetch_bio(f, u, app)
                            else:
                                bio = ''

                        additional_info: List[str]|None = None
                        if args.add_additional_info:
                            additional_info = get_additional_info(u)
                        
                        write_row(writer, u, is_member, bio, additional_info)
                        io.CSV_FLUSHED = False  
                        
                        users[int(u.id)] = u
                        total_messages += 1
                        last_msg_id = m.id
                        curr_messages += 1

                        if total_messages >= args.messages_limit:
                            break
                        
                except errors.FloodWait as e:
                    await io.flood_wait_or_exit(f, int(getattr(e, 'value', 0)), 
                                                'fetching members from messages')
                except errors.RPCError as e:
                    io.exit_on_rpc(f, e, 'fetching members from messages')
                except Exception as e:
                    io.message(f, 'error', 'UNEXPECTED_ERROR',
                               when='fetching members from messages',
                               error=str(e))
                
                if curr_messages == 0 or last_msg_id is None or last_msg_id <= 1:
                    break
                
                offset_id = last_msg_id
                
        await _write_members_from_messages()
        io.message(None, 'info', 'MEMBERS_FROM_MESSAGES_FETCHED', 
                   total=total_messages)
        
    io.CSV_FLUSHED = True
    

async def main():
    io.message(None, 'info', 'SCRIPT_STARTED', script='get_members.py')
    try:
        args = parse_args()
    except Exception as e:
        io.message(None, 'error', "ARGPARSE_ERROR", error=str(e))
        
    if args.limit > 50000:
        io.message(None, 'error', 'MEMBERS_LIMIT_TOO_HIGH', 
                limit=args.limit, max=50000)
    
    if args.parse_from_messages and args.messages_limit > 5000:
        io.message(None, 'error', 'MESSAGE_LIMIT_TOO_HIGH',
                limit=args.messages_limit, max=5000)
    
    if not args.output:
        args.output = f'get-members-{int(time.time())}.csv'
    
    options: Dict[str, Any] = get_tdlib_options()
    api_id: int = options["api_id"]
    api_hash: str = options["api_hash"]    
    
    kind, name = parse_chat_name(args.chat)
    if kind == ChatNameKind.EMPTY:
        io.message(None, 'error', 'INVALID_CHAT_NAME', name=args.chat)

    if kind == ChatNameKind.INVITE_LINK:
        io.message(None, 'error', 'INVITE_LINK_NOT_SUPPORTED', name=args.chat)
        
    columns = ['user_id', 'username', 'first_name', 'last_name', 'is_member', 'is_bot']
    if args.parse_bio:
        columns.append('bio')
    if args.add_additional_info:
        columns.extend(['premium_status', 'is_deleted', 'is_scam', 'is_verified', 'phone_number'])
    with open(args.output, 'w', newline='', encoding='utf-8') as f:
        writer = csv.writer(f)
        writer.writerow(columns)
    io.CSV_FLUSHED = True

    async with Client(SESSION, api_id, api_hash) as app:
        users: Dict[int, types.User] = await fetch_members(app, name, args)
    
        if args.parse_from_messages:
            await fetch_members_from_messages(app, name, args, users)
    
        io.message(None, 'info', 'ALL_DONE', total=len(users), output=os.path.abspath(args.output))
        
        
if __name__ == '__main__':
    try:
        asyncio.run(main())
    except Exception as e:
        io.message(None, 'error', 'UNEXPECTED_ERROR', 
                   when='main', error=str(e))