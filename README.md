# ephemera-link

Simple web app for creating encrypted secrets that can be viewed only once via unique random link.

## Features

- Unique link to a secret with a configurable length.
- Simple API for creating and retrieving secrets.
- The data can be stored in memory or in a database file.
- The data is encrypted with the AES-256 algorithm.
- System light/dark theme.
- Russian language support.

## Configuration

Can be passed via `.env` file in the same folder where app binary is located.

Or via environment variables.

| Variable | Default value | Description |
|----------|---------------|-------------|
| URI | `http://localhost:8080/` | Root URI with schema for links |
| LISTEN_PORT | `8080` | On which TCP port app should listen |
| KEY_PART | | Secret string for encryption (`pwgen -cns 24 1`) |
| PERSISTENT_STORAGE | `false` | Bool: store data in memory or in database file |
| STORAGE_PATH | `data` | Path to database folder |
| ID_LENGTH | `8` | First random part of the link |
| KEY_LENGTH | `8` | Second random part of the link |
| RUN_CLEARING_INTERVAL | `1800` | How often clear expired secrets in seconds |
| SECRETS_EXPIRE | `86400` | Secret expire time after created in seconds |

Sample `.env` file:

```sh
DOMAIN=https://secret.example.com/
LISTEN_PORT=8080
KEY_PART='Gmd6QO6W1wjEDcugEbBGPcXS'
PERSISTENT_STORAGE=true
STORAGE_PATH=/app/data
ID_LENGTH=8
KEY_LENGTH=8
RUN_CLEARING_INTERVAL=1800
SECRETS_EXPIRE=86400
```

## API

Create secret `/api/create`:

```sh
$ echo -n "some secret text here" | curl -sSX POST http://localhost:8080/api/create --data-binary @- | jq
{
 "expires_at": "2025-07-13T15:55:17+03:00",
 "expires_in_seconds": "86400",
 "id": "gzF3tLzP",
 "key": "CZOzcJU2",
 "link": "http://localhost:8080/c/gzF3tLzP/CZOzcJU2"
}
```

Retrieve secret `/api/retrieve`:

```sh
# With id and key
$ echo -e '{
  "id": "gzF3tLzP",
  "key": "CZOzcJU2"
}' | curl -sSX POST http://localhost:8080/api/retrieve --data-binary @-
some secret text here

# Or with link
$ curl -sSX POST http://localhost:8080/api/retrieve -d '{"link": "http://localhost:8080/c/gzF3tLzP/CZOzcJU2"}'
some secret text here
```

## Build and run

```bash
docker build -t ephemera-link .
docker run --name ephemera-link -d -p 8080:8080 --env-file .env -v data:/app/data ephemera-link
```

## Screenshots

<details>
  <summary>Light theme</summary>

Index page:

![Index page](screenshots/light/index.png "Index page")

Saved page:

![Saved page](screenshots/light/saved.png "Saved page")

View page:

![View page](screenshots/light/view.png "View page")

Retrieve page:

![Retrieve page](screenshots/light/retrieve.png "Retrieve page")

</details>

<details>
  <summary>Dark theme</summary>

Index page:

![Index page](screenshots/dark/index.png "Index page")

Saved page:

![Saved page](screenshots/dark/saved.png "Saved page")

View page:

![View page](screenshots/dark/view.png "View page")

Retrieve page:

![Retrieve page](screenshots/dark/retrieve.png "Retrieve page")

</details>
