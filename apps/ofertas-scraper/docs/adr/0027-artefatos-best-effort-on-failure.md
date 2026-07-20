---
status: accepted
---

# Artefatos are best-effort per attempt; no placeholders for skipped steps

A processing attempt should retain whatever debug material it actually produced (PDF, images, Extrator raw, validated result). On hard failure, later steps are simply absent from that attempt directory — we do not invent empty `extrator-raw.json` / `validated.json` (or empty PDFs) just to keep a four-file layout. Full four-file attempts remain the happy path. Rejected: requiring all four always (noise and fake transcripts) and skipping Artefatos entirely when download fails (hides early failures when a tentativa folder would still help ops).
