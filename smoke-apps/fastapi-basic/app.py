from fastapi import FastAPI

app = FastAPI()


@app.get("/")
def home():
    return "fastapi-basic smoke app"
