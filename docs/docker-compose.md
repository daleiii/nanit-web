# Docker compose quick guide

When running the app as a service, it is useful to use docker-compose for easier configuraiton.

Create `docker-compose.yml` file somewhere on your machine.

Adjust following example (see [.env.sample](../.env.sample) for configuration options).

```yaml
version: '3.8'
services:
  nanit-web:
    # Image to pull, adjust the :suffix for your version tag
    image: deltathreed/nanit-web:latest
    container_name: nanit-web
    # Makes the container auto-start whenever you restart your computer
    restart: unless-stopped
    # Expose the web dashboard and RTMP ports
    ports:
    - "8080:8080"
    - "1935:1935"
    volumes:
    - nanit-data:/data
    # Configuration (see .env.sample file for all the options)
    # Note: Credentials are configured via the web dashboard at http://localhost:8080
    environment:
    - "NANIT_HTTP_PORT=8080"
    - "NANIT_RTMP_ADDR=xxx.xxx.xxx.xxx:1935"
    - "NANIT_SESSION_FILE=/data/session.json"
    - "NANIT_LOG_LEVEL=info"

volumes:
  nanit-data:
```

## Control the app container

Run in the same directory as your `docker-compose.yml` file

```bash
# Start the app
docker-compose up -d

# See the logs (Use Ctrl+C to terminate)
docker-compose logs -f nanit-web

# Stop the app
docker-compose stop

# Upgrade the app (ie. after you have changed the version tag or to pull fresh dev image)
docker-compose pull  # pulls fresh image
docker-compose down  # removes previously created container
docker-compose up -d # creates new container with fresh image (do this after every change in the docker-compose file)

# Uninstall the app
docker-compose down
```
