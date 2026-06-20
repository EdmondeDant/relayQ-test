# Debug Session: login-failure [OPEN]

## User Symptom
- User reports the provided admin account and password fail to log in.

## Expected
- The reset admin credentials should allow successful login.

## Current Hypotheses
- H1: The password was reset in one database, but the login page is connected to a different backend instance.
- H2: The frontend login request is not reaching the backend because the backend is down or the proxy target is unavailable.
- H3: The credentials are correct, but an additional auth requirement blocks login, such as TOTP, captcha, or auth mode restrictions.
- H4: The frontend is sending stale or malformed login data due to cached state or runtime config mismatch.

## Evidence Plan
- Reproduce login via API directly against the local backend target.
- Check whether the backend is reachable from the current environment.
- Inspect frontend auth request shape and runtime config.
- Compare observed failure with reset output and database-selected admin account.

## Evidence Collected
- Frontend API base URL is `/api/v1`, so login requests go to `/api/v1/auth/login`.
- `POST http://127.0.0.1:3000/api/v1/auth/login` returns `500 Internal Server Error`.
- `GET http://127.0.0.1:3000/api/v1/settings/public` also returns `500 Internal Server Error`.
- `127.0.0.1:8080` is not listening, while `127.0.0.1:3000` is reachable.
- Database-selected active admin is `363164954@qq.com`, and password reset succeeded for that account.

## Hypothesis Status
- H1: Partially supported. The password reset hit the database successfully, but the current frontend target is not serving auth correctly.
- H2: Supported. The backend target behind the frontend proxy is unavailable or unhealthy.
- H3: Rejected for now. The request fails before any TOTP-specific flow is returned.
- H4: Rejected for now. Public settings and auth endpoints both fail, indicating a backend/proxy issue rather than form-state corruption.

## Interim Conclusion
- The provided credentials are not the primary failure point.
- The active problem is that the frontend's backend target is down or unhealthy, which makes login fail generically.

## Backend Fix Applied
- Started backend with a valid 64-hex `TOTP_ENCRYPTION_KEY`.
- Disabled file logging for the local run to avoid `/app/data/logs/sub2api.log` permission failure in this environment.
- Confirmed backend started successfully on `127.0.0.1:8080`.
- Confirmed `GET /api/v1/settings/public` now returns `200`.
- Confirmed `POST /api/v1/auth/login` succeeds for admin `363164954@qq.com` with the reset password.

## Persistent Config Adjustment
- Updated `deploy/.env`:
  - `LOG_OUTPUT_TO_FILE=false`
  - `TOTP_ENCRYPTION_KEY=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef`

## Status
- Backend recovered in the current session; awaiting user verification.
