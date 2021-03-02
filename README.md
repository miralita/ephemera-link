# ephemera-link

One-time links (simple secret pusher)

## Description

The data is stored in memory or in a database file (see .env.sample for details).

The data is encrypted with the AES-256 algorithm.

The encryption key consists of a common part stored in an environment variable and a unique part. A unique part is generated for each secret and passed in the link.
The link also contains a unique identifier for the secret. 

## Build and run

1. Prepare .env-file based on .env.sample

2. Create "data" directory for persistent storage

```bash
docker build -t ephemera-link .
docker run --name ephemera-link -d -p 8834:8834 --env-file .env -v data:/app/data ephemera-link
```
