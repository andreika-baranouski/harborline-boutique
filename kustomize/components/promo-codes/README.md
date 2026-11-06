# Enable promo codes at checkout

By default the cart page has no promo code input, and the frontend drops any posted `promo_code`.
This component sets `ENABLE_PROMO_CODES=true` on the `frontend` Deployment, which shows a
"Promo code" input with an "Apply" button on the cart page and passes the code to `checkoutservice`.

The code table itself lives in the `PROMO_CODES` environment variable of `checkoutservice`
(JSON; see `base/checkoutservice.yaml`):

```json
[{"code":"WELCOME10","percent":10},
 {"code":"HARBOR25","percent":25,"end":"2026-12-31"}]
```

`percent` is 1-100; `start` and `end` are optional, inclusive, `YYYY-MM-DD` in UTC. Unknown, expired
or not-yet-active codes never block an order: it is placed at full price and the confirmation page
says why. Deploy `checkoutservice` with the new code before turning this component on.

## Deploy with Kustomize

From the `kustomize/` folder at the root level of this repository, execute this command:

```bash
kustomize edit add component components/promo-codes
```

You can locally render these manifests by running `kubectl kustomize .` as well as deploying them by running `kubectl apply -k .`.
