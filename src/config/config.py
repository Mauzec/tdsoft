import os
from typing import Dict, Any
from dotenv import load_dotenv

load_dotenv()

def get_tdlib_options() -> Dict[str, Any]:
    try:
        api_id = int(os.getenv("API_ID"))
        api_hash = os.getenv("API_HASH")
    except ValueError:
        raise ValueError("API_ID must be an integer")
    if not api_id or not api_hash:
        raise ValueError("API_ID and API_HASH must be set in environment")
    return {
        "api_id": api_id,
        "api_hash": api_hash
    }