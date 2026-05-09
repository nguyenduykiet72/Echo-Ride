#!/usr/bin/env bash
# Configures APISIX routes for echo-ride. Re-running is safe (PUT is idempotent).
#
# Auth model:
#   * /api/v1/auth/{register,login,refresh}  -> public (no JWT)
#   * /api/v1/auth/logout                    -> JWT required
#   * /api/v1/users/*, /api/v1/admin/users/* -> JWT required, X-User-Id/Role injected
#   * /api/v1/rides/*                        -> JWT required, X-User-Id/Role injected
#   * /ws                                    -> JWT required (location-service WS)
#
# Requires: bash, curl, jq.
#
# Usage:
#   ADMIN_KEY=edd1c9f034335f136f87ad84b625c8f1 \
#   APISIX_ADMIN=http://localhost:9180/apisix/admin \
#   JWT_SECRET=mysecretkey \
#   ./apisix_admin_setup.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROUTES_DIR="${SCRIPT_DIR}/routes"
INJECT_HEADERS_LUA="${SCRIPT_DIR}/inject_user_headers.lua"

ADMIN_KEY="${ADMIN_KEY:-edd1c9f034335f136f87ad84b625c8f1}"
APISIX_ADMIN="${APISIX_ADMIN:-http://localhost:9180/apisix/admin}"
JWT_SECRET="${JWT_SECRET:-mysecretkey}"

# Hosts/ports as seen from inside the apisix container.
# host.docker.internal works because docker-compose sets extra_hosts on apisix.
export AUTH_HOST="${AUTH_HOST:-host.docker.internal}"
export AUTH_PORT="${AUTH_PORT:-8114}"
export USER_HOST="${USER_HOST:-host.docker.internal}"
export USER_PORT="${USER_PORT:-8115}"
export RIDE_HOST="${RIDE_HOST:-host.docker.internal}"
export RIDE_PORT="${RIDE_PORT:-8111}"
export LOCATION_HOST="${LOCATION_HOST:-host.docker.internal}"
export LOCATION_PORT="${LOCATION_PORT:-8112}"
export JWT_SECRET

INJECT_HEADERS_FN="$(cat "${INJECT_HEADERS_LUA}")"

put() {
    local path="$1"; local body="$2"
    curl -sS --fail-with-body -X PUT "${APISIX_ADMIN}${path}" \
        -H "X-API-KEY: ${ADMIN_KEY}" \
        -H 'Content-Type: application/json' \
        -d "$body" >/dev/null
    echo "  PUT ${path}"
}

render() {
    # Substitute ${VAR} placeholders from env, then replace __INJECT_HEADERS__
    # with the Lua function content (escaped via jq).
    local file="$1"
    local rendered
    rendered="$(envsubst < "$file")"
    jq --arg fn "$INJECT_HEADERS_FN" \
        '(.. | strings) |= sub("__INJECT_HEADERS__"; $fn)' \
        <<<"$rendered"
}

echo ">> Consumer: echo_ride"
put /consumers "$(render "${ROUTES_DIR}/00-consumer-echo_ride.json")"

echo ">> Upstreams"
put /upstreams/auth-service     "$(render "${ROUTES_DIR}/10-upstream-auth.json")"
put /upstreams/user-service     "$(render "${ROUTES_DIR}/11-upstream-user.json")"
put /upstreams/ride-service     "$(render "${ROUTES_DIR}/12-upstream-ride.json")"
put /upstreams/location-service "$(render "${ROUTES_DIR}/13-upstream-location.json")"

echo ">> Routes"
put /routes/auth-public  "$(render "${ROUTES_DIR}/20-route-auth-public.json")"
put /routes/auth-logout  "$(render "${ROUTES_DIR}/21-route-auth-logout.json")"
put /routes/users        "$(render "${ROUTES_DIR}/30-route-users.json")"
put /routes/admin-users  "$(render "${ROUTES_DIR}/31-route-admin-users.json")"
put /routes/rides        "$(render "${ROUTES_DIR}/40-route-rides.json")"
put /routes/ws-location  "$(render "${ROUTES_DIR}/50-route-ws-location.json")"

echo ">> APISIX configuration complete."
