# Debug Session: image-url-hang

Status: [OPEN]

## Bug Summary

线上客户通过外部客户端 WorkBuddy 调用 RelayQ `/v1/images/generations` 生成 `gpt-image-2` 图片时，请求默认或显式 `response_format: "url"` 会挂死直到超时；RelayQ 网页后台模型测试可以出图，因为测试页强制 `response_format: "b64_json"`。

## Source Report

- File: `c:\work\RelayQ-test\bug记录排查\BUG-REPORT.md`
- Key evidence from report:
  - `response_format: "b64_json"` returns HTTP 200 in ~40s.
  - `response_format: "url"` or omitted response_format hangs with HTTP 000.
  - Web model test sends `response_format: "b64_json"`.

## Hypotheses

1. H1: The gateway forwards `response_format: "url"` unchanged, and upstream hangs or returns a URL response that RelayQ waits on indefinitely.
2. H2: RelayQ response handling downloads/proxies image URLs when response format is `url`, and that URL fetch blocks without a strict timeout.
3. H3: Usage/billing or content moderation tries to parse URL-format image response differently and blocks before returning to client.
4. H4: Request-body rewrite/defaulting does not normalize omitted `response_format`, so external clients default to the broken `url` path while internal model test avoids it.
5. H5: The issue is not in the upstream call but in client response streaming/connection handling after receiving upstream response.

## Planned Evidence Points

- Log parsed incoming image request: model, response_format, body size, multipart flag.
- Log forward payload response_format and upstream model before sending upstream request.
- Log upstream request start/end/error with duration and status code.
- Log whether any URL response is downloaded/transformed after upstream response.
- Log response finalization path and duration.

## Timeline

- Created session file and hypotheses.
- Read `backend/internal/service/openai_images.go`.
- Evidence from code: non-streaming images response handling reads upstream JSON and writes it with `c.Data`; no image URL download/proxy/cache path exists in this handler.
- Evidence from report: `response_format: "b64_json"` succeeds; `response_format: "url"` or omitted hangs against the configured upstream.
- Interim conclusion: H2 is rejected for this gateway path; H1/H4 are supported. Minimal fix should normalize upstream images requests to `b64_json` and adapt downstream response when caller requested/defaulted to URL.
- Fix implemented in `backend/internal/service/openai_images.go`:
  - For JSON image requests, upstream `response_format` is forced to `b64_json` when the caller requests `url` or omits `response_format`.
  - For callers that requested/defaulted to URL, the non-stream response adapter converts upstream `data[].b64_json` into `data[].url` data URIs and removes `b64_json` from the returned items.
- Regression tests added in `backend/internal/service/openai_images_test.go`:
  - `TestConvertOpenAIImagesB64JSONToDataURL`
  - `TestBuildOpenAIImagesForwardBodyNormalizesURLToB64JSON`
  - `TestBuildOpenAIImagesForwardBodyDefaultsToB64JSON`
- Verification: `go test ./internal/service -run "Test(ConvertOpenAIImagesB64JSONToDataURL|BuildOpenAIImagesForwardBody|OpenAIGatewayServiceParseOpenAIImagesRequest)"` passed.

## Conclusion

- Confirmed root cause direction: external clients hit the upstream URL/default response path, while RelayQ's web model test avoids it by forcing `b64_json`.
- RelayQ does not download URL-format images in this non-stream gateway path, so the hang is not caused by local URL proxy/download logic.
- The minimal safe fix is to always use the known-good upstream `b64_json` path for URL/default callers, then return OpenAI-compatible `url` fields as data URIs to preserve client compatibility.
