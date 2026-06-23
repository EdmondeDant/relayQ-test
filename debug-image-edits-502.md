# Debug Session: image-edits-502
- **Status**: [FIXED]
- **Issue**: `gpt-image-2 /v1/images/edits` passed group permission and reached upstream, but returned `502 Upstream request failed`.
- **Debug Server**: cleaned
- **Log File**: cleaned

## Reproduction Steps
1. Start backend service on local port `3000`.
2. Use the image-group API key that can see `gpt-image-2` in `GET /v1/models`.
3. Send `POST /v1/images/edits` with prompt, image, mask, and image edit options.
4. Observe gateway response `502 Upstream request failed`.

## Hypotheses & Verification
| ID | Hypothesis | Likelihood | Effort | Evidence |
|----|------------|------------|--------|----------|
| A | The gateway builds an invalid `responses/editing` body for `edits`, so upstream rejects it. | High | Low | Pending |
| B | The selected upstream image account does not support the `responses/editing` edit shape even though `gpt-image-2` is listed. | High | Medium | Pending |
| C | Upstream returns a detailed JSON error, but gateway wraps it into a generic 502 and loses the real reason. | High | Low | Pending |
| D | The mapping of `images` / `mask` / `input_fidelity` is incompatible with the chosen upstream station. | Medium | Medium | Pending |
| E | The upstream response is valid but the gateway fails while translating `responses` output back into images output. | Medium | Medium | Pending |

## Log Evidence
- `A @ openai_images_responses.go:1130`: request body is built as `responses` editing JSON with `tool.action=edit`, `tool.model=gpt-image-2`, `input_image_mask` present, but request body `stream=true` while client request `stream=false`.
- `E @ openai_images_responses.go:731`: upstream responds with `200`, `Content-Type: text/event-stream`, and body contains:
  - `response.created`
  - `response.output_text.delta` with markdown image link
  - `response.completed`
- The `response.completed` payload does **not** contain `image_generation_call.result`; instead it contains `output[0].type=message` and `content[0].type=output_text` with a generated image URL.
- Client still receives `502 Upstream request failed`, so failure happens after upstream success, during response parsing / response shape adaptation.

## Verification Conclusion
- Hypothesis A: **Partially confirmed / not root cause**. Request body shape is valid enough to reach upstream and get a `200`.
- Hypothesis B: **Rejected for current account**. Selected account `生图gpt` at `https://image.codesonline.dev/v1` does answer the request.
- Hypothesis C: **Rejected**. The failure is not an upstream HTTP error body being hidden; upstream success body was captured.
- Hypothesis D: **Rejected for current reproduction**. Mask/image/options do not block upstream processing.
- Hypothesis E: **Confirmed**. Upstream returns a successful SSE body, but in a markdown-link/output_text shape that current image parser does not treat as image output, leading to downstream failure.

## Final Fix
- Added fallback parsing for markdown image links carried in `response.output_text.delta` / `response.completed`.
- Downloaded the generated image bytes from the extracted URL and rebuilt a standard OpenAI Images response with `data[].b64_json`.
- Fixed the `responses/editing` top-level model selection for image edits so image-only upstream accounts do not fail with a hard-coded orchestrator model.
- Re-ran real `POST /v1/images/edits` verification after the fix and confirmed the gateway returns `200` with `created` and `data[].b64_json`.

## Cleanup
- Removed `backend/internal/service/debug_image_edits_502.go`.
- Removed debug instrumentation from `backend/internal/service/openai_images_responses.go`.
- Deleted `.dbg/trae-debug-log-image-edits-502.ndjson`.
