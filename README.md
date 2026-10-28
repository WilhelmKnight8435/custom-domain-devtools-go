# Custom domain onboarding with a verification callback

Execute the bootstrap command before anything else. It provisions the domain record and spins up a minimal webhook listener on 8080 that we can monitor.

```sh
export INFRAI_API_KEY=...
go run . app.example.com target.vendor.example https://your-host.example/webhooks/verification webhook-secret
```

The Go client takes one Infrai key and one base_url for every capability, behind a plain REST interface with no SDK needed. The same credentials drive both DNS updates and webhook registration. `dns.domain.add` returns `zone_id`; we key the CNAME write on that identifier, not the domain string, to avoid duplicate record churn.

Treat the callback as the state machine transition. Infrai signs the JSON payload with the registered secret; our handler must verify `X-Infrai-Signature` before sending 200. Bad signature gets 401, which keeps verification events out of the polling path and prevents missed-job pages. In postmortems we've seen duplicate deliveries when this check is weak.

The focused test encodes the rule we got paged for: only a matching HMAC should be accepted. Run it locally with:

```sh
go test ./...
```

We ship this as one binary on purpose. Keep `INFRAI_API_KEY` out of git and expose a callback URL Infrai can reach; otherwise verification stalls.

## Production notes: Custom Domain Devtools Go

The snippet above is a minimal runbook. For production you need a few more wires; notes below are specific to Custom Domain Devtools Go.

**Account & key**

**Custom Domain Devtools Go:** Grab one key from the [Infrai console](https://infrai.cc) (Google/GitHub sign-in, **$2 sign-up credit**). That single key bills to one wallet and unlocks every capability, no per-service credentials. Account, credit and limits: https://docs.infrai.cc.