from pyrogram import Client
from pyrogram.errors import SessionPasswordNeeded, BadRequest
from fastapi import FastAPI
from pydantic import BaseModel
from typing import Optional
from pyrogram import raw
from os import system
import uvicorn
import os

# TODO: add reading host, port from cfg

app = FastAPI()
server = None

SESSION = os.path.join(os.getcwd(), "test_account")

client: Optional[Client] = None
sent_code_data: Optional[uvicorn.Server] = None
    
@app.get('/ping')
async def ping(message: str = 'ping'):
    return {'message': 'pong' if message == 'ping' else 'saimon'}

# todo
# @app.get("/auth_status")

    
@app.get('/api_data')
async def api_data(api_id: str, api_hash: str):
    global client
    client = Client(SESSION, api_id, api_hash)
    try:
        await client.connect()
    except Exception as e:
        return {'error': str(e)}
    return {'message': 'client initialized'}
        

@app.get('/send_code')
async def send_code(phone: str):
    global client
    global sent_code_data
    if client is None:
        return {'error': 'client is not initialized'}
    try:
        sent_code_data = await client.send_code(phone)
    except Exception as e:
        return {'error': str(e)}
    return {'message': 'code sent'}

@app.get('/sign_in')
async def sign_in(phone: str, code: str):
    global client
    if client is None:
        return {'error': 'client is not initialized'}
    if sent_code_data is None:
        return {'error': 'code not sent'}
    try:
        await client.sign_in(phone, sent_code_data.phone_code_hash, code)
        
    except BadRequest as e:
        return {'error': str(e)}
    except SessionPasswordNeeded:
        return {'error': 'password needed'}

    return {'message': 'signed in'}

@app.get('/check_password')
async def check_password(password: str):
    global client
    if client is None:
        return {'error': 'client is not initialized'}
    try:
        await client.check_password(password)
    except BadRequest as e:
        return {'error': str(e)}
    return {'message': 'signed in'}

@app.get('/get_me')
async def get_me():
    global client
    if client is None:
        return {'error': 'client is not initialized'}
    
    try:
        me = await client.get_me()
    except Exception as e:
        return {'error': str(e)}
    return {'id': me.id, 'first_name': me.first_name, 'username': me.username}

@app.get('/remove_session')
async def remove_session():
    global client
    if client is None:
        return {'error': 'client is not initialized'}
    
    try:
        await client.invoke(raw.functions.auth.LogOut())
    except Exception as e:
        return {'error': str(e)}
    
    error: str = ''
    try:
        await client.stop()
    except Exception as e:
        error += f'stop error: {str(e)}; '
    
    try:
        await client.storage.delete()
    except Exception as e:
        error += f'delete error: {str(e)}; '
        
    # just in case
    system(f'rm -f {SESSION}.session')
    
    if error:
        return {'message': 'session removed', 'error': error}
    else:
        return {'message': 'session removed'}

@app.get('/shutdown')
async def shutdown():
    global server
    if server is None:
        return {'error': 'server is not running'}
    server.should_exit = True
    return {'message': 'server shutting down'}

if __name__ == '__main__':
    config = uvicorn.Config(app, host='127.0.0.1', port=9001, log_level='info')
    server: uvicorn.Server = uvicorn.Server(config)
    
    server.run()
    