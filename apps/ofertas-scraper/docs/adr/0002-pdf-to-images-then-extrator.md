# PDF to images before Extrator

We do not send PDF bytes to the Extrator. Each Documento is rasterized to page images, downscaled (default max edge 1280px), and only those images are sent, because Gemini vision works on images, prompt caching stays on the stable system prompt, and Artefatos can retain exactly what the model saw for debug and reprocessing.
