# frontend

The frontend serves the storefront web UI and calls the backend services over gRPC.
Dependencies are managed with Go modules (`go.mod`).

## Environment variables

### Required

| Variable | Description |
|---|---|
| `PRODUCT_CATALOG_SERVICE_ADDR`, `CURRENCY_SERVICE_ADDR`, `CART_SERVICE_ADDR`, `RECOMMENDATION_SERVICE_ADDR`, `CHECKOUT_SERVICE_ADDR`, `SHIPPING_SERVICE_ADDR`, `AD_SERVICE_ADDR`, `SHOPPING_ASSISTANT_SERVICE_ADDR` | `host:port` of each backend service. The frontend panics at startup if one is unset. |
| `COLLECTOR_SERVICE_ADDR` | OpenTelemetry collector address. Required only when `ENABLE_TRACING=1`. |

### Optional

| Variable | Accepted values | Default | Effect |
|---|---|---|---|
| `ENABLE_RECOMMENDATIONS` | on: `true`, `1`, `on`, `yes`; off: `false`, `0`, `off`, `no` (case-insensitive) | on | When off, the product, cart and order pages render without the "You may also like" strip and the frontend sends no requests to the recommendation service. Unrecognised values keep it on and log a warning. The state is logged at startup (`Recommendations enabled.` / `Recommendations disabled.`); changing it needs a pod restart. |
| `ENABLE_TRACING` | `1` | off | Exports traces to `COLLECTOR_SERVICE_ADDR`. |
| `ENABLE_PROFILER` | `1` | off | Enables Cloud Profiler. |
| `CYMBAL_BRANDING` | `true` | off | Uses the Cymbal Shops branding. |
| `ENABLE_ASSISTANT` | `true` | off | Shows the shopping assistant. |
| `ENABLE_SINGLE_SHARED_SESSION` | `true` | off | All visitors share one session (and cart). |
| `FRONTEND_MESSAGE` | any text | empty | Banner message shown on every page. |
| `PACKAGING_SERVICE_URL` | URL | empty | Optional packaging service; packaging info is shown on the product page when set. |
| `BASE_URL` | path prefix | empty | Serves the site under a path prefix. |
| `PORT` | port | `8080` | HTTP listen port. |
