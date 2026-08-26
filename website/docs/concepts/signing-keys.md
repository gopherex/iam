---
id: signing-keys
title: Signing keys & token profiles
sidebar_label: Signing keys
---

# Signing keys & token profiles

Every access token and id_token IAM issues is an **RS256 JWT** signed with a
per-project, per-environment RSA-2048 key. Relying parties verify it against the
published JWK Set:

```
{public_url}/p/{project_id}/e/{environment}/.well-known/jwks.json
```

The private PEM is encrypted at rest with `IAM_SERVICE_AUTH_ENCRYPTION_KEY` —
which is why that value must stay stable forever. Lose it and no existing key can
be read.

## The key lifecycle

| Status | Signs new tokens | Published in JWKS |
| --- | --- | --- |
| `active` | yes — exactly one per environment | yes |
| `inactive` | no | yes |
| `retired` | no | **no** |

`inactive` is what makes a safe rotation possible: a new key can be published
and picked up by every relying party's JWKS cache before it starts signing
anything.

## Rotating

```bash
# 1. generate a new key, published but not yet signing
curl -sX POST https://auth.example.com/v1/projects/prj_7Fk2/admin/jwks/rotate \
  -H "Authorization: Bearer $ADMIN_TOKEN" -H "X-Environment: live" \
  -H "Content-Type: application/json" -d '{"activate": false}'
# -> { "key": { "kid": "…", "alg": "RS256", "status": "inactive" } }

# 2. wait out your relying parties' JWKS cache (minutes, not seconds)

# 3. promote it
curl -sX POST .../admin/jwks/{kid}/activate -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Environment: live"

curl -s .../admin/jwks -H "Authorization: Bearer $ADMIN_TOKEN" -H "X-Environment: live"
```

Passing `{"activate": true}` to `rotate` does both steps at once — convenient in
`test`, wrong in `live` if any client caches JWKS.

:::warning Activation retires the previous key immediately
Promoting a key sets the old one to `retired`, and a retired key leaves the JWK
Set. Tokens already signed with it stop verifying — so a rotation invalidates
outstanding access tokens rather than letting them expire. The blast radius is
one `access_ttl` (default 10 minutes); refresh tokens are unaffected, so clients
recover on their next refresh. Rotate when that gap is acceptable, or shorten
`access_ttl` first.
:::

`DELETE .../admin/jwks/{kid}` removes a key permanently. Only ever delete one
that is already `retired`.

Keys are per **environment**: rotating in `test` does not touch `live`.

## Token profiles

A token profile is a named set of token settings for one audience, and a client
is bound to at most one. That is the shape the question usually has: one relying
party needs a five-minute token with its own `aud`, another needs an hour.

```bash
curl -sX POST .../admin/token-profiles -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"api","audience":"https://api.example.com","access_ttl":600,
       "claims_template":{"tier":"standard"}}'
# -> { "profile": { "id": "tp_api", ... } }

# bind a client to it
curl -sX PATCH .../admin/apps/app_web -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" -d '{"token_profile_id":"tp_api"}'

# render what it would produce for a given user
curl -sX POST .../admin/token-profiles/tp_api/preview -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" -d '{"user_id":"usr_9"}'
```

Tokens minted for a bound client then carry the profile's `aud`, its lifetimes
and its template claims. Anything the profile leaves unset stays as the
project's [`session_policy`](/guides/admin-config), so a profile that only
shortens a lifetime does only that. An unbound client is unaffected, and a
binding pointing at a deleted profile falls back to the defaults rather than
failing the mint.

`claims_template` is a static map — the same values for everybody the profile
applies to. A claim the provider owns (`sub`, `iss`, `aud` beyond the profile's
own, `exp`, `jti`, `sid`, `scope`, `groups`, …) is ignored rather than
overwritten: letting a template rewrite `sub` would be an impersonation
primitive. Anything that varies per user belongs in
[roles and the `groups` claim](/concepts/oidc-federation) instead.
