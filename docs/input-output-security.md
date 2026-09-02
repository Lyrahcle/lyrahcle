# Input/output security contracts

Lyrahcle keeps two regression contracts in CI for SQL construction and browser XSS sinks.

## SQL

Store values must be passed to PostgreSQL through pgx parameters. Dynamic SQL identifiers are forbidden unless the construction is explicitly security-reviewed and selected from an internal whitelist. `internal/store/space_cves.go` is currently the only reviewed exception: its filter values, pagination values, and dates remain parameters, while the sort expression and direction are selected from fixed code-owned values.

`python3 scripts/check-sql-construction.py .` fails CI if another store introduces SQL string formatting without review. The `spaceCVESortClause` unit tests also include malicious sort strings to ensure they fall back to the default expression rather than entering the query.

## Browser XSS

React text interpolation is the default rendering path. Direct DOM HTML assignment (`innerHTML`, `outerHTML`, `document.write`) and `dangerouslySetInnerHTML` are forbidden in application source. The world map SVG is generated locally from bundled static data, URI-encoded, and rendered through an `<img>` source instead of being parsed into the application DOM as raw HTML.

Every `target="_blank"` anchor must contain both `noopener` and `noreferrer`. TOTP setup links are rendered only when their URL scheme is exactly `otpauth:`; HTTP(S), `javascript:`, `data:`, protocol-relative, malformed, and empty values are rejected.

Run the browser contract with:

```sh
python3 -m unittest scripts/test_check_frontend_xss.py
python3 scripts/check-frontend-xss.py .
```
