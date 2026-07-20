# Upstash Redis with SRH for local development

We persist Fontes, Documentos, Ofertas, and Falhas de Extração in Upstash Redis and talk to it over the Upstash REST API so local and cloud use the same client. Locally we run Redis plus Serverless Redis HTTP (SRH) as documented by Upstash, instead of a plain Redis TCP client or a non-existent official “Upstash image,” so `UPSTASH_REDIS_REST_URL` / `UPSTASH_REDIS_REST_TOKEN` work unchanged against SRH or Upstash cloud.
