import grpc, os, sys
from contextlib import asynccontextmanager
from fastapi import FastAPI, Depends, HTTPException
from google.protobuf.json_format import MessageToDict
from fastapi.middleware.cors import CORSMiddleware

sys.path.append(os.path.join(os.path.dirname(__file__), "generated"))

from generated import search_pb2_grpc, search_pb2
from server import serve_grpc

grpc_server = None
grpc_channel = None

@asynccontextmanager
async def lifespan(app: FastAPI):
    global grpc_server, grpc_channel

    grpc_server = await serve_grpc()
    grpc_channel = grpc.aio.insecure_channel('localhost:50051')

    print("Startup: gRPC Server running and Channel connected.")

    yield

    print("Shutdown: Closing gRPC connections...")
    if grpc_channel:
        await grpc_channel.close()
    if grpc_server:
        await grpc_server.stop(grace=5)

app = FastAPI(lifespan=lifespan)

app.add_middleware(
    CORSMiddleware,
    # Allow your Vite frontend (localhost:5173) to connect
    allow_origins=["http://localhost:5173", "http://127.0.0.1:5173"],
    allow_credentials=True,
    allow_methods=["*"],  # Allow all methods (GET, POST, etc.)
    allow_headers=["*"],  # Allow all headers
)

def get_search_stub():
    if not grpc_channel:
        raise HTTPException(status_code=503, detail="gRPC Channel not initialized")
    return search_pb2_grpc.SearchEngineStub(channel=grpc_channel)

@app.get("/autocomplete")
async def autocomplete_endpoint(
    q: str,
    limit: int = 10,
    stub: search_pb2_grpc.SearchEngineStub = Depends(get_search_stub)
):
    try:
        request = search_pb2.AutocompleteRequest(prefix=q, limit=limit)
        grpc_response = await stub.Autocomplete(request)

        return MessageToDict(grpc_response, preserving_proto_field_name=True)

    except grpc.RpcError as e:
        raise HTTPException(status_code=500, detail=f"gRPC Error: {e.details()}")

@app.get("/search")
async def search_endpoint(
    q: str,
    stub: search_pb2_grpc.SearchEngineStub = Depends(get_search_stub)
):
    try:
        request = search_pb2.SearchRequest(query=q)
        grpc_response = await stub.Search(request)
        return MessageToDict(grpc_response, preserving_proto_field_name=True)
    except grpc.RpcError as e:
        raise HTTPException(status_code=500, detail=f"gRPC Error: {e.details()}")
