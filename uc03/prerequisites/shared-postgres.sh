#!/usr/bin/env bash
# Platform-owned shared PostgreSQL for the EXISTING definition
# postgres-shared-staging. It exists before any deployment and is never created,
# changed or destroyed by the IDP; the IDP only reads its connection record
# from the Secret Store (idpsecret://platform/shared-postgres-staging).
set -euo pipefail
cd "$(dirname "$0")/.."
NAME=idp-shared-postgres
PASSWORD_FILE=var/platform/shared-postgres.password
mkdir -p var/platform && chmod 700 var/platform

if ! docker inspect "$NAME" >/dev/null 2>&1; then
  [ -f "$PASSWORD_FILE" ] || (head -c 24 /dev/urandom | od -An -tx1 | tr -d ' \n' > "$PASSWORD_FILE")
  chmod 600 "$PASSWORD_FILE"
  docker run -d --name "$NAME" --restart unless-stopped --network kind \
    -e POSTGRES_USER=reports -e POSTGRES_DB=reports -e POSTGRES_PASSWORD="$(cat "$PASSWORD_FILE")" \
    postgres:17-alpine >/dev/null
  echo "started $NAME"
fi
for i in $(seq 1 30); do
  docker exec "$NAME" pg_isready -U reports >/dev/null 2>&1 && break
  sleep 1
done
HOST=$(docker inspect -f '{{with index .NetworkSettings.Networks "kind"}}{{.IPAddress}}{{end}}' "$NAME")
RECORD=$(mktemp)
trap 'rm -f "$RECORD"' EXIT
printf '{"host":"%s","port":"5432","database":"reports","username":"reports","password":"%s"}' "$HOST" "$(cat "$PASSWORD_FILE")" > "$RECORD"
./bin/idp secret-put platform/shared-postgres-staging "$RECORD"
echo "shared postgres reachable from kind clusters at $HOST:5432"
