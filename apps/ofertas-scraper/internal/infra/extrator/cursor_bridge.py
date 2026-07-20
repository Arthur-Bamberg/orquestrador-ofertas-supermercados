#!/usr/bin/env python3
"""Cursor Agent SDK bridge for the Go Extrator adapter (ADR 0034).

Reads one JSON request from stdin, writes one JSON response to stdout.
Keeps Cursor SDK / local agent runtime out of the Go module graph.
"""

from __future__ import annotations

import base64
import json
import os
import re
import sys
import tempfile
from typing import Any


def _fail(message: str, code: str = "error") -> None:
    json.dump({"ok": False, "error": {"code": code, "message": message}}, sys.stdout)
    sys.stdout.write("\n")
    sys.stdout.flush()
    raise SystemExit(0)


def _extract_json_object(text: str) -> dict[str, Any]:
    text = (text or "").strip()
    if not text:
        raise ValueError("empty model response")
    fence = re.search(r"```(?:json)?\s*([\s\S]*?)```", text, re.IGNORECASE)
    if fence:
        text = fence.group(1).strip()
    start = text.find("{")
    end = text.rfind("}")
    if start < 0 or end <= start:
        raise ValueError("no JSON object in model response")
    return json.loads(text[start : end + 1])


def _usage_dict(usage: Any) -> dict[str, int]:
    if usage is None:
        return {
            "promptTokens": 0,
            "cacheTokens": 0,
            "outputTokens": 0,
        }
    return {
        "promptTokens": int(getattr(usage, "input_tokens", 0) or 0),
        "cacheTokens": int(getattr(usage, "cache_read_tokens", 0) or 0),
        "outputTokens": int(getattr(usage, "output_tokens", 0) or 0),
    }


def main() -> None:
    try:
        req = json.load(sys.stdin)
    except Exception as exc:  # noqa: BLE001
        _fail(f"invalid stdin json: {exc}", "invalid_request")

    api_key = (req.get("apiKey") or os.environ.get("CURSOR_API_KEY") or "").strip()
    if not api_key:
        _fail("CURSOR_API_KEY required", "invalid_request")

    model = (req.get("model") or "composer-2.5").strip()
    model_params = req.get("modelParams") or [{"id": "fast", "value": "true"}]
    prompt = (req.get("prompt") or "").strip()
    schema_json = (req.get("schemaJSON") or "").strip()
    pages = req.get("pages") or []
    if not prompt:
        _fail("prompt required", "invalid_request")
    if not pages:
        _fail("pages required", "invalid_request")

    try:
        from cursor_sdk import (  # type: ignore
            Agent,
            LocalAgentOptions,
            ModelParameterValue,
            ModelSelection,
            RateLimitError,
            SDKImage,
        )
    except Exception as exc:  # noqa: BLE001
        _fail(
            f"cursor-sdk import failed (pip install cursor-sdk): {exc}",
            "dependency",
        )

    selection: Any = model
    try:
        params = tuple(
            ModelParameterValue(id=str(p["id"]), value=str(p["value"]))
            for p in model_params
            if isinstance(p, dict) and "id" in p and "value" in p
        )
        selection = ModelSelection(id=model, params=params)
    except Exception:
        selection = model

    user_prefix = (
        "SYSTEM PROMPT (follow strictly):\n"
        f"{prompt}\n\n"
        "OUTPUT JSON SCHEMA (respond with one object matching this contract):\n"
        f"{schema_json}\n\n"
        "Do not use tools. Do not edit files. Reply with ONLY the JSON object "
        '(root {"ofertas":[...]}). No markdown fences, no commentary.\n\n'
    )

    all_ofertas: list[Any] = []
    uso_paginas: list[dict[str, Any]] = []
    prompt_tokens = 0
    cache_tokens = 0
    output_tokens = 0

    tmp_dir = tempfile.mkdtemp(prefix="cursor-extrator-")
    try:
        with Agent.create(
            model=selection,
            api_key=api_key,
            local=LocalAgentOptions(cwd=tmp_dir),
        ) as agent:
            for page in pages:
                page_num = int(page.get("page") or 0)
                if page_num <= 0:
                    page_num = 1
                b64 = page.get("jpegBase64") or ""
                if not b64:
                    continue
                try:
                    jpeg = base64.b64decode(b64)
                except Exception as exc:  # noqa: BLE001
                    _fail(f"page {page_num}: bad jpegBase64: {exc}", "invalid_request")
                img_path = os.path.join(tmp_dir, f"page-{page_num}.jpg")
                with open(img_path, "wb") as fh:
                    fh.write(jpeg)

                user = (
                    user_prefix
                    + f"TASK: Extraia TODAS as ofertas com preço legível desta única "
                    f"página (página {page_num}) do encarte. Não omita itens da grade; "
                    f"não amostrar."
                )
                try:
                    run = agent.send(
                        {
                            "text": user,
                            "images": [SDKImage.from_file(img_path)],
                        }
                    )
                    result = run.wait()
                except RateLimitError as exc:
                    _fail(str(exc), "rate_limit")
                except Exception as exc:  # noqa: BLE001
                    name = type(exc).__name__
                    msg = str(exc).lower()
                    code = "error"
                    if "rate" in name.lower() or "429" in msg or "resource_exhausted" in msg:
                        code = "rate_limit"
                    elif "unavailable" in msg or "timeout" in msg or "deadline" in msg:
                        code = "unavailable"
                    _fail(f"page {page_num}: {exc}", code)

                text = result.result if isinstance(result.result, str) else str(result.result or "")
                try:
                    parsed = _extract_json_object(text)
                except Exception as exc:  # noqa: BLE001
                    _fail(f"page {page_num}: decode structured output: {exc}", "decode")

                ofertas = parsed.get("ofertas")
                if ofertas is None:
                    ofertas = []
                if not isinstance(ofertas, list):
                    _fail(f"page {page_num}: ofertas is not a list", "decode")
                all_ofertas.extend(ofertas)

                u = _usage_dict(getattr(result, "usage", None))
                prompt_tokens += u["promptTokens"]
                cache_tokens += u["cacheTokens"]
                output_tokens += u["outputTokens"]
                uso_paginas.append(
                    {
                        "page": page_num,
                        "promptTokens": u["promptTokens"],
                        "cacheTokens": u["cacheTokens"],
                        "outputTokens": u["outputTokens"],
                    }
                )
    finally:
        try:
            for name in os.listdir(tmp_dir):
                try:
                    os.remove(os.path.join(tmp_dir, name))
                except OSError:
                    pass
            os.rmdir(tmp_dir)
        except OSError:
            pass

    json.dump(
        {
            "ok": True,
            "ofertas": all_ofertas,
            "uso": {
                "model": model,
                "promptTokens": prompt_tokens,
                "cacheTokens": cache_tokens,
                "outputTokens": output_tokens,
                "paginas": uso_paginas,
            },
        },
        sys.stdout,
        ensure_ascii=False,
    )
    sys.stdout.write("\n")
    sys.stdout.flush()


if __name__ == "__main__":
    main()
