"""HTTP MITM Proxy para capturar y contar tokens de APIs de AI.

Uso:
    python proxy.py --port 8080 --model gpt-4o --jaula plain
    # Luego configura HTTP_PROXY=http://localhost:8080 HTTPS_PROXY=http://localhost:8080
"""

from __future__ import annotations

import argparse
import http.server
import json
import logging
import os
import time
import urllib.request
import urllib.error
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

from token_counter import TokenCounter

logging.basicConfig(
    level=logging.INFO,
    format="[%(asctime)s] %(message)s",
    datefmt="%H:%M:%S",
)
logger = logging.getLogger("proxy")


class BenchmarkProxyHandler(http.server.BaseHTTPRequestHandler):
    """Handler HTTP que captura requests a APIs de AI."""

    # Compartido entre instancias
    counter: TokenCounter | None = None
    jaula: str = ""
    output_dir: Path = Path("logs")
    target_host: str = "api.anthropic.com"
    start_time: float = 0.0

    def do_POST(self):
        """Captura POST requests."""
        content_length = int(self.headers.get("Content-Length", 0))
        body = self.rfile.read(content_length) if content_length > 0 else b""

        # Leer metadata de la request
        host = self.headers.get("Host", self.target_host)
        path = self.path
        req_body_str = body.decode("utf-8", errors="replace")

        # Reenviar a API real
        target_url = f"https://{host}{path}"
        req = urllib.request.Request(
            target_url,
            data=body,
            headers={k: v for k, v in self.headers.items() if k.lower() not in ("host", "content-length", "proxy-connection")},
            method="POST",
        )

        try:
            with urllib.request.urlopen(req, timeout=120) as resp:
                resp_body = resp.read()
                resp_body_str = resp_body.decode("utf-8", errors="replace")

                # Contar tokens
                if self.counter:
                    self.counter.log_request("user", req_body_str, {
                        "host": host, "path": path, "jaula": self.jaula
                    })
                    self.counter.log_request("assistant", resp_body_str, {
                        "host": host, "path": path, "jaula": self.jaula
                    })

                # Responder al cliente
                self.send_response(resp.status)
                for k, v in resp.headers.items():
                    if k.lower() not in ("content-encoding", "content-length", "transfer-encoding"):
                        self.send_header(k, v)
                self.send_header("Content-Length", str(len(resp_body)))
                self.end_headers()
                self.wfile.write(resp_body)

                elapsed = time.time() - self.start_time
                summary = self.counter.summary() if self.counter else {}
                logger.info(
                    f"[{self.jaula}] {resp.status} | "
                    f"req={len(req_body_str)}b resp={len(resp_body_str)}b | "
                    f"tokens={summary.get('total_tokens', '?')} | "
                    f"turns={summary.get('turns', '?')} | "
                    f"elapsed={elapsed:.1f}s"
                )

        except urllib.error.HTTPError as e:
            self.send_response(e.code)
            self.end_headers()
            self.wfile.write(e.read())
            logger.error(f"[{self.jaula}] HTTP {e.code}: {e.reason}")
        except Exception as e:
            self.send_response(502)
            self.end_headers()
            self.wfile.write(f'{{"error":"proxy error: {e}"}}'.encode())
            logger.error(f"[{self.jaula}] Proxy error: {e}")

    def do_GET(self):
        """Ignora GETs (health checks, etc)."""
        self.send_response(200)
        self.end_headers()
        self.wfile.write(b'{"status":"proxy running"}')

    def log_message(self, format, *args):
        """Silencia logs HTTP por defecto."""
        pass


class BenchmarkProxy:
    """Proxy HTTP que captura y mide tokens de APIs de AI."""

    def __init__(self, port: int = 8080, model: str = "gpt-4o", jaula: str = "unknown"):
        self.port = port
        self.jaula = jaula
        self.counter = TokenCounter(model)
        self.output_dir = Path(f"logs/benchmark-{jaula}")
        self.output_dir.mkdir(parents=True, exist_ok=True)

        # Configurar handler
        BenchmarkProxyHandler.counter = self.counter
        BenchmarkProxyHandler.jaula = jaula
        BenchmarkProxyHandler.output_dir = self.output_dir
        BenchmarkProxyHandler.start_time = time.time()

        self.server = http.server.HTTPServer(
            ("127.0.0.1", port),
            BenchmarkProxyHandler,
        )

    def start(self):
        """Inicia el proxy (bloqueante)."""
        logger.info(f"🚀 Proxy iniciado en 127.0.0.1:{self.port} para jaula '{self.jaula}'")
        logger.info(f"   Modelo: {self.counter.model} | Método: {self.counter.method}")
        logger.info(f"   Configura: HTTP_PROXY=http://127.0.0.1:{self.port}")
        logger.info(f"   Logs: {self.output_dir}/")
        try:
            self.server.serve_forever()
        except KeyboardInterrupt:
            self.stop()

    def stop(self):
        """Detiene el proxy y guarda resultados."""
        logger.info(f"\n⏹  Proxy detenido. Guardando resultados...")
        self.server.shutdown()

        # Guardar log
        data_path = self.output_dir / "data.json"
        self.counter.save(str(data_path))

        summary = self.counter.summary()
        logger.info(f"📊 Resumen [{self.jaula}]:")
        logger.info(f"   Tokens input:  {summary['input_tokens']:>8,}")
        logger.info(f"   Tokens output: {summary['output_tokens']:>8,}")
        logger.info(f"   Total:         {summary['total_tokens']:>8,}")
        logger.info(f"   Turns:         {summary['turns']:>8,}")
        logger.info(f"   Guardado en:   {data_path}")


def main():
    parser = argparse.ArgumentParser(description="Benchmark proxy MITM")
    parser.add_argument("--port", type=int, default=8080, help="Puerto del proxy")
    parser.add_argument("--model", default="gpt-4o", help="Modelo a usar (gpt-4o, claude-sonnet-4)")
    parser.add_argument("--jaula", required=True, help="Nombre de la jaula (plain, gentle, zyro)")
    args = parser.parse_args()

    proxy = BenchmarkProxy(port=args.port, model=args.model, jaula=args.jaula)
    proxy.start()


if __name__ == "__main__":
    main()
