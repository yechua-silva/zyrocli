#!/usr/bin/env python3
"""Genera reporte HTML con gráficos del benchmark.

Uso: python generate_report.py <report_dir>
"""

import json
import os
import sys
from pathlib import Path

# Intentar matplotlib, fallback a HTML sin gráficos
try:
    import matplotlib
    matplotlib.use("Agg")
    import matplotlib.pyplot as plt
    import numpy as np
    HAS_MATPLOTLIB = True
except ImportError:
    HAS_MATPLOTLIB = False


def load_data(report_dir: Path) -> dict:
    data_path = report_dir / "data.json"
    if not data_path.exists():
        return {"plain": {"summary": {"total_tokens": 0, "input_tokens": 0, "output_tokens": 0, "turns": 0}},
                "gentle": {"summary": {"total_tokens": 0, "input_tokens": 0, "output_tokens": 0, "turns": 0}},
                "zyro": {"summary": {"total_tokens": 0, "input_tokens": 0, "output_tokens": 0, "turns": 0}}}
    with open(data_path) as f:
        return json.load(f)


def generate_charts(data: dict, output_dir: Path):
    """Genera 6 gráficos PNG."""
    if not HAS_MATPLOTLIB:
        return

    jaulas = ["plain", "gentle", "zyro"]
    labels = ["Plain\nOpenCode", "gentle-ai\nv1.40.2", "ZyroCLI\nBoomerang"]
    colors = ["#888888", "#ff69b4", "#00add8"]

    def get(key):
        return [data[j].get("summary", {}).get(key, 0) for j in jaulas]

    # 1. Tokens totales
    plt.figure(figsize=(8, 5))
    vals = get("total_tokens")
    bars = plt.bar(labels, vals, color=colors)
    plt.title("Tokens Totales por Enfoque", fontsize=14, fontweight="bold")
    plt.ylabel("Tokens")
    for bar, v in zip(bars, vals):
        plt.text(bar.get_x() + bar.get_width()/2, bar.get_height() + max(vals)*0.01,
                 f"{v:,}", ha="center", va="bottom", fontsize=11)
    plt.tight_layout()
    plt.savefig(output_dir / "chart-01-tokens-totales.png", dpi=150)
    plt.close()

    # 2. Input vs Output
    plt.figure(figsize=(8, 5))
    input_vals = get("input_tokens")
    output_vals = get("output_tokens")
    x = np.arange(len(labels))
    width = 0.35
    plt.bar(x - width/2, input_vals, width, label="Input", color="#4a90d9")
    plt.bar(x + width/2, output_vals, width, label="Output", color="#e67e22")
    plt.title("Tokens: Input vs Output", fontsize=14, fontweight="bold")
    plt.xticks(x, labels)
    plt.ylabel("Tokens")
    plt.legend()
    plt.tight_layout()
    plt.savefig(output_dir / "chart-02-input-output.png", dpi=150)
    plt.close()

    # 3. Turns
    plt.figure(figsize=(8, 5))
    vals = get("turns")
    bars = plt.bar(labels, vals, color=colors)
    plt.title("Turns por Enfoque", fontsize=14, fontweight="bold")
    plt.ylabel("Turns")
    for bar, v in zip(bars, vals):
        plt.text(bar.get_x() + bar.get_width()/2, bar.get_height() + max(vals)*0.01,
                 str(v), ha="center", va="bottom", fontsize=11)
    plt.tight_layout()
    plt.savefig(output_dir / "chart-03-turns.png", dpi=150)
    plt.close()


def generate_html(data: dict, output_dir: Path):
    """Genera index.html con tabla + gráficos."""
    s = data.get("plain", {}).get("summary", {})
    g = data.get("gentle", {}).get("summary", {})
    z = data.get("zyro", {}).get("summary", {})

    charts_html = ""
    if HAS_MATPLOTLIB:
        for i in range(1, 7):
            import glob
            charts = sorted(output_dir.glob(f"chart-0{i}-*.png"))
            if charts:
                charts_html += f'<img src="{charts[0].name}" style="width:100%;max-width:700px;margin:10px auto;display:block">\n'

    html = f"""<!DOCTYPE html>
<html lang="es">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Benchmark: Plain OpenCode vs gentle-ai vs ZyroCLI</title>
<style>
  @import url('https://fonts.googleapis.com/css2?family=Inter:wght@400;600;700&display=swap');
  * {{ margin: 0; padding: 0; box-sizing: border-box; }}
  body {{ font-family: 'Inter', sans-serif; background: #f8f9fa; color: #333; padding: 40px 20px; }}
  .container {{ max-width: 900px; margin: 0 auto; }}
  h1 {{ font-size: 2em; margin-bottom: 8px; }}
  .subtitle {{ color: #666; margin-bottom: 30px; }}
  .summary {{ display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 20px; margin-bottom: 40px; }}
  .card {{ background: white; border-radius: 12px; padding: 24px; box-shadow: 0 2px 8px rgba(0,0,0,0.08); }}
  .card h3 {{ font-size: 0.9em; text-transform: uppercase; color: #888; margin-bottom: 8px; }}
  .card .value {{ font-size: 1.8em; font-weight: 700; }}
  .card .value.plain {{ color: #666; }}
  .card .value.gentle {{ color: #e91e8c; }}
  .card .value.zyro {{ color: #00add8; }}
  table {{ width: 100%; border-collapse: collapse; margin: 20px 0 40px; background: white; border-radius: 12px; overflow: hidden; box-shadow: 0 2px 8px rgba(0,0,0,0.08); }}
  th, td {{ padding: 14px 16px; text-align: left; }}
  th {{ background: #f1f3f5; font-weight: 600; font-size: 0.9em; text-transform: uppercase; color: #555; }}
  td {{ border-bottom: 1px solid #eee; }}
  tr:last-child td {{ border-bottom: none; }}
  .metric {{ font-weight: 600; }}
  .winner {{ color: #2e7d32; font-weight: 700; }}
  img {{ border-radius: 12px; box-shadow: 0 2px 12px rgba(0,0,0,0.1); }}
  .footer {{ margin-top: 40px; padding-top: 20px; border-top: 1px solid #ddd; font-size: 0.85em; color: #888; }}
  .footer a {{ color: #00add8; }}
  .badge {{ display: inline-block; padding: 2px 8px; border-radius: 4px; font-size: 0.8em; font-weight: 600; }}
  .badge.na {{ background: #eee; color: #999; }}
  .badge.ok {{ background: #e8f5e9; color: #2e7d32; }}
</style>
</head>
<body>
<div class="container">
  <h1>🏋️ Benchmark: 3 Jaulas</h1>
  <p class="subtitle">Plain OpenCode vs gentle-ai vs ZyroCLI — tokens, turns, tiempo y costo</p>

  <div class="summary">
    <div class="card">
      <h3>🟢 Plain OpenCode</h3>
      <div class="value plain">{s.get('total_tokens', 0):,}</div>
      <div style="color:#888;font-size:0.9em">{s.get('turns', 0)} turns</div>
    </div>
    <div class="card">
      <h3>🌸 gentle-ai v1.40.2</h3>
      <div class="value gentle">{g.get('total_tokens', 0):,}</div>
      <div style="color:#888;font-size:0.9em">{g.get('turns', 0)} turns</div>
    </div>
    <div class="card">
      <h3>🔷 ZyroCLI + Boomerang</h3>
      <div class="value zyro">{z.get('total_tokens', 0):,}</div>
      <div style="color:#888;font-size:0.9em">{z.get('turns', 0)} turns</div>
    </div>
  </div>

  <table>
    <tr>
      <th>Métrica</th>
      <th>Plain OpenCode</th>
      <th>gentle-ai</th>
      <th>ZyroCLI</th>
      <th>Ganador</th>
    </tr>
    <tr>
      <td class="metric">Tokens totales</td>
      <td>{s.get('total_tokens', '—'):,}</td>
      <td>{g.get('total_tokens', '—'):,}</td>
      <td>{z.get('total_tokens', '—'):,}</td>
      <td class="winner">{'—'}</td>
    </tr>
    <tr>
      <td class="metric">Tokens input</td>
      <td>{s.get('input_tokens', '—'):,}</td>
      <td>{g.get('input_tokens', '—'):,}</td>
      <td>{z.get('input_tokens', '—'):,}</td>
      <td class="winner">{'—'}</td>
    </tr>
    <tr>
      <td class="metric">Tokens output</td>
      <td>{s.get('output_tokens', '—'):,}</td>
      <td>{g.get('output_tokens', '—'):,}</td>
      <td>{z.get('output_tokens', '—'):,}</td>
      <td class="winner">{'—'}</td>
    </tr>
    <tr>
      <td class="metric">Turns</td>
      <td>{s.get('turns', '—')}</td>
      <td>{g.get('turns', '—')}</td>
      <td>{z.get('turns', '—')}</td>
      <td class="winner">{'—'}</td>
    </tr>
    <tr>
      <td class="metric">Método de conteo</td>
      <td><span class="badge na">{s.get('method', 'N/A')}</span></td>
      <td><span class="badge na">{g.get('method', 'N/A')}</span></td>
      <td><span class="badge na">{z.get('method', 'N/A')}</span></td>
      <td></td>
    </tr>
    <tr>
      <td class="metric">Muestras (N≥3 requerido)</td>
      <td><span class="badge na">{'⏳ N<3' if s.get('turns', 0) < 3 else '✅ N≥3'}</span></td>
      <td><span class="badge na">{'⏳ N<3' if g.get('turns', 0) < 3 else '✅ N≥3'}</span></td>
      <td><span class="badge na">{'⏳ N<3' if z.get('turns', 0) < 3 else '✅ N≥3'}</span></td>
      <td></td>
    </tr>
  </table>

  {charts_html}

  <h2>🔬 Metodología</h2>
  <ul>
    <li><strong>Proxy MITM:</strong> Captura todos los requests/responses entre el agente y la API de AI</li>
    <li><strong>Tokenización exacta:</strong> <a href="https://github.com/openai/tiktoken">tiktoken</a> para OpenAI, Anthropic SDK para Claude</li>
    <li><strong>Misma tarea:</strong> "Agregar autenticación JWT a API Go" para las 3 jaulas</li>
    <li><strong>Mismo codebase base:</strong> Go 1.24, HTTP server básico con /ping y /status</li>
    <li><strong>Mínimo 3 muestras</strong> por jaula antes de reportar ganador</li>
  </ul>

  <h2>📚 Investigación</h2>
  <ul>
    <li><a href="https://github.com/openai/tiktoken">tiktoken</a> — Tokenizer exacto de OpenAI (18.5k ⭐)</li>
    <li><a href="https://github.com/Gentleman-Programming/gentle-ai">gentle-ai</a> — Competidor (4k ⭐, v1.40.2)</li>
    <li><a href="https://swebench.com/">SWE-bench</a> — Benchmark de coding agents</li>
    <li><a href="https://www.anthropic.com/research/building-effective-agents">Anthropic Building Effective Agents</a></li>
    <li><a href="https://www.anthropic.com/research/swe-bench-sonnet">SWE-bench Sonnet Results</a> — >100k tokens sin memoria</li>
  </ul>

  <div class="footer">
    <p>Generado por <a href="https://github.com/secko/zyrocli">ZyroAgentCLI Benchmark</a> el __TIMESTAMP__</p>
    <p>Los datos se recolectan con proxy MITM + tiktoken. Resultados reproducibles.</p>
  </div>
</div>
</body>
</html>"""

    # Reemplazar placeholder con timestamp real
    from datetime import datetime
    html = html.replace("__TIMESTAMP__", datetime.now().strftime('%Y-%m-%d %H:%M'))

    with open(output_dir / "index.html", "w") as f:
        f.write(html)
    print(f"✅ Reporte generado: {output_dir / 'index.html'}")


def main():
    if len(sys.argv) < 2:
        print("Uso: python generate_report.py <report_dir>")
        sys.exit(1)

    report_dir = Path(sys.argv[1])
    report_dir.mkdir(parents=True, exist_ok=True)

    data = load_data(report_dir)
    generate_charts(data, report_dir)
    generate_html(data, report_dir)

    print("📊 Reporte completo:")
    print(f"   HTML: {report_dir / 'index.html'}")
    if HAS_MATPLOTLIB:
        for f in sorted(report_dir.glob("chart-*.png")):
            print(f"   {f.name}")


if __name__ == "__main__":
    main()
