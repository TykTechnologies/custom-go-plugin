#!/usr/bin/env bash
./tyk/scripts/wait-for-it.sh -t 300 localhost:3000
sleep 1;
status=$(curl -s -o /dev/null -w "%{http_code}" localhost:3000)

if [ "302" == "$status" ] || [ "200" == "$status" ]; then
  source .env

  # Bootstrap Tyk dashboard with default organisation.
  curl -s -X POST localhost:3000/bootstrap \
    --data "owner_name=$ORG" \
    --data "owner_slug=$SLUG" \
    --data "email_address=$EMAIL" \
    --data "first_name=$FIRST" \
    --data "last_name=$LAST" \
    --data "password=$PASSWORD" \
    --data "confirm_password=$PASSWORD" \
    --data "terms=on"

  # Get organisation ID.
  ORG=$(curl -s -X GET localhost:3000/admin/organisations \
    --header "admin-auth: 12345" | \
    jq -r '.organisations[0].id')

  # Create a new admin user and get user access token.
  TOKEN=$(curl -s -X POST localhost:3000/admin/users \
    --header "admin-auth: 12345" \
    --data "{
      \"org_id\": \"$ORG\",
      \"first_name\": \"Admin\",
      \"last_name\": \"User\",
      \"email_address\": \"admin@tyk.io\",
      \"active\": true,
      \"user_permissions\": { \"IsAdmin\": \"admin\" }
    }" | \
    jq -r '.Message')

  # Create httpbin API
  curl -s -X POST localhost:3000/api/apis/oas \
    --header "authorization: $TOKEN" \
    --header "Content-Type: application/json" \
    --data "@tyk/scripts/oas.json" > /dev/null
fi
