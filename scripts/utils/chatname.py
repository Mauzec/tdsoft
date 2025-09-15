from typing import Tuple
import regex as re
from enum import Enum, auto
from urllib.parse import urlparse

class ChatNameKind(Enum):
    def _generate_next_value_(self, *args):
        return self.lower()
    EMPTY = auto()
    'Invalid/empty chat name'
    
    USERNAME = auto()
    INVITE_LINK = auto()
    CHAT_ID = auto()
    
    def __repr__(self):
        return f'mauzec.{self}'
    

        
def parse_chat_name(name: str) -> Tuple[ChatNameKind, str]:
    '''
    Determines whether the given string is a TG-link(invite or direct), or username, or chat Id.\n
    It can't determine if username, chat_id, or invite link is valid or not, just the format.
    There are only 'light' checks.
    
    Returns
    - (EMPTY, '') if the input is invalid.
    - (USERNAME, username) if the input is a username or a t.me/username link
    - (INVITE_LINK, t.me/...) if the input is an invite link
    - (CHAT_ID, chat_id) if the input is a chat id (integer)
    '''
    if not isinstance(name, str):
        return (ChatNameKind.EMPTY, '')
    
    name = name.strip()
    if not name:
        return (ChatNameKind.EMPTY, '')
    
    # is chat id?
    if name[0] == '0':
        return (ChatNameKind.EMPTY, '')
    
    digits = name[1:] if name[0] == '-' else name
    if not digits:
        return (ChatNameKind.EMPTY, '')
    if digits.isdigit():
        if digits[0] == '0' or len(digits) > 18:
            return (ChatNameKind.EMPTY, '')
        return (ChatNameKind.CHAT_ID, name)
        
    
    # is username?
    username = name[1:] if name.startswith('@') else name
    if not username:
        return (ChatNameKind.EMPTY, '')
    if re.fullmatch(r'[A-Za-z0-9_]{2,32}', username):
        return (ChatNameKind.USERNAME, username)
    if name.startswith('@'):
        return (ChatNameKind.EMPTY, '')
    
    # is link?

    if '://' not in name:
        name = 'https://' + name

    try:
        url = urlparse(name)
    except Exception:
        return (ChatNameKind.EMPTY, '')

    host = (url.hostname or '').lower()
    if host.startswith('www.'):
        host = host[4:]

    
    if host != 't.me':
        return (ChatNameKind.EMPTY, '')

    path = (url.path or '').lstrip('/')
    if not path:
        return (ChatNameKind.EMPTY, '')

    # invite links: +hash or joinchat/hash
    if path.startswith('+'):
        token = path[1:]
        if not token or '/' in token:
            return (ChatNameKind.EMPTY, '')
        return (ChatNameKind.INVITE_LINK, token)

    if path.startswith('joinchat/'):
        token = path[len('joinchat/'):]
        if not token or '/' in token:
            return (ChatNameKind.EMPTY, '')
        return (ChatNameKind.INVITE_LINK, token)

    if '/' in path:
        if path.endswith('/'):
            path = path.rstrip('/')
        else:
            return (ChatNameKind.EMPTY, '')

    if re.fullmatch(r'[A-Za-z0-9_]{2,32}', path):
        return (ChatNameKind.USERNAME, path)

    return (ChatNameKind.EMPTY, '')
    