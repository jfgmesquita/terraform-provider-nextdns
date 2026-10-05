# CLAUDE.md

Terraform provider for [NextDNS](https://nextdns.io/), built on [terraform-plugin-framework](https://github.com/hashicorp/terraform-provider-scaffolding-framework). Published as `jfgmesquita/nextdns`. Not affiliated with NextDNS.

## Commands

```shell
make build     # go build
make test      # unit tests (no network, no account)
make generate  # format examples/ and regenerate docs/
make testacc   # acceptance tests: real API, needs NEXTDNS_API_KEY of a test-only account
golangci-lint run
```

CI (`.github/workflows/test.yml`) runs build, lint, `make generate` with a diff check, and all tests. Acceptance tests are skipped when `NEXTDNS_API_KEY` is not set.

## Layout

- `internal/nextdns/`: API client. `client.go` has `Do` (JSON, `X-Api-Key`, errors as `APIError`, no redirects to other hosts); one file per API section. Knows nothing about Terraform.
- `internal/provider/`: the provider (`provider.go`), one file per resource or data source, shared helpers (`lists.go`, `notes.go`).
- `examples/`: hand-written HCL used in the docs. `templates/`: doc page templates. `docs/`: generated, never edit by hand.

## Design rules

- Settings and list resources (security, privacy, settings, parental control, denylist, allowlist, rewrites) manage the whole section: anything left out of the configuration is reset to its default or removed. Create and Update send the whole section.
- Their `Delete` does not call the API; it only removes the resource from the state. `nextdns_profile`'s `Delete` deletes the profile.
- They are imported by profile ID and use `profilePartNotes` in their description.
- `Read` removes the resource from the state on HTTP 404.
- Every attribute has a `MarkdownDescription`. Descriptions use the official wording of the NextDNS dashboard where it exists.
- Lists of IDs are sets. Lists of objects with one defaulted attribute are set blocks; with several defaulted attributes, use a map keyed by ID (the framework applies all defaults in a set element when only one is set).
- Examples target enterprise use (office or team profiles).
- After changing a description or example, run `make generate` and commit `docs/`.

## API behaviour

The official docs (https://nextdns.github.io/api/) are incomplete. Verified behaviour:

- Errors can come with any status, including HTTP 200 with an `errors` list. Validation errors point to the field with `source.pointer`.
- `POST /profiles` returns the ID in `fingerprint`; `GET` returns the real fingerprint.
- `PATCH` on a section only changes the fields sent. An invalid value rejects the whole request.
- `PUT /security/tlds` empties the list on an invalid TLD; `PATCH /security` with `tlds` does not, so use it.
- Denylist and allowlist are replaced with `PUT`; NextDNS reorders them.
- Rewrites cannot be replaced at once: add with `POST`, remove with `DELETE` by ID. Add before deleting.
- Parental control returns HTTP 500 (after almost a minute) when the recreation schedule is sent together with services and categories: send it in its own request. Updates use three requests to fail closed.
- Recreation times are stored as `HH:MM:SS`; the schedule replaces the previous one; an empty schedule has a `null` timezone.
- Settings: `bav` is bypass age verification; log `drop` fields are the inverse of the dashboard's "log" options; log locations are `us`, `eu` and `ch`.
- Setup: the `dnscrypt` field is a DNS stamp of the DNS-over-HTTPS endpoint.
- Public catalogs (no API key): `/privacy/blocklists`, `/privacy/natives`, `/parentalcontrol/services`, `/parentalcontrol/categories`.
