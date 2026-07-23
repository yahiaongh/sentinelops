"""Configuration loaded from environment variables."""

from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    database_url: str = (
        "postgresql://sentinelops:devpassword@localhost:5432/sentinelops"
    )
    ollama_base_url: str = "http://host.docker.internal:11434"
    ollama_model: str = "llama3.2:3b"
    ollama_keep_alive: str = "10m"
    ollama_num_predict: int = 300
    ollama_timeout_seconds: float = 180.0
    default_lookback_hours: int = 6
    metrics_port: int = 9300

    class Config:
        env_prefix = ""


settings = Settings()
