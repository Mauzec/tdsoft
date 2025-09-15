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

# TODO:

'''
Search messages in a group/channel/private
'''

SESSION = os.path.join(os.getcwd(), "test_account")

def parse_args() -> argparse.Namespace:
    p = argparse.ArgumentParser(
        description='''Search messages in a group/channel/private chat by keywords. 
        WARNING: without --limit-history it will read all history of the chat. ''')
    
    p.add_argument(
        'username', help='username of the user', required=True)
    p.add_argument(
        "chat", help="username, t.me/username, id", required=True)
    
    p.add_argument(
        "--output", type=str, default=f'search-messages-{int(time.time())}.csv',
        help="output csv file path; default is search-messages-<time>.csv")
    
    p.add_argument(
        'from_date', type=str, help='date from which to start searching messages, format DD.MM.YYYY',
        required=True)
    p.add_argument(
        'to_date', type=str, help='date to which to search messages, format DD.MM.YYYY',
        required=True)
    
    # keywords?
    
    p.parse_args()
    

async def main():
    pass


if __name__ == '__main__':
    args = parse_args()
    asyncio.run(main())
    
    