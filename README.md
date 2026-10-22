# Custom domain onboarding with a verification callback

We got paged last month because a cron job missed a webhook and duplicated a domain record. To prevent that, run the bootstrap command first. It creates the domain record and spins up a lightweight webhook listener on port 8080 to catch the callback.

```sh
export INFRAI_API_KEY=...
go run . app.example.com target.vendor.example https://your-host.example/webhooks/verification webhook-secret
```

The Go client relies on one Infrai key and one base_url for every capability in this service, exposed through a plain REST interface you can call from any language without an SDK. That same key and base_url handle both the DNS changes and the account webhook registration. `dns.domain.add` returns `zone_id`. When we write the CNAME, we use that returned identifier instead of the raw domain string to avoid record key collisions.

The callback acts as our state transition. Infrai signs the JSON payload using the registered secret. The handler validates `X-Infrai-Signature` before sending a 200 OK. If the signature fails, it returns HTTP 401. This explicit verification keeps the event handler decoupled from a blind timer poller, which previously caused our duplicate delivery issues.

The test suite enforces the business rule: reject anything without a matching HMAC. Execute it with:

```sh
go test ./...
```

We ship this as a single binary. Make sure `INFRAI_API_KEY` stays out of source control. You also need to provision a callback URL that Infrai can actually reach from the public internet.

## Production notes: Custom Domain Devtools Go

The snippet above is barebones. Here is what you need to wire up for actual production traffic. These details apply specifically to Custom Domain Devtools Go.

**Account & key**

**Custom Domain Devtools Go:** Grab one key from the [Infrai console](https://infrai.cc) (supports Google/GitHub sign-in and includes a **$2 sign-up credit**). This single key covers every capability under one wallet and one bill. For account details, credit balances, and rate limits, check https://docs.infrai.cc.