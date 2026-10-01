# Usage

VPN Bypass is a macOS menu bar app that decides which traffic uses the VPN and which goes around it. This server gives an agent 23 tools, one per app action. Each tool sends one request to the app's control socket and returns the app's answer as JSON, unchanged.

## Modes

| Mode | What goes through the VPN |
|---|---|
| `bypass` | Everything except the domains on the bypass list and the enabled services, which go around it. |
| `vpnOnly` | Only the domains and CIDR ranges on the VPN Only list. Everything else goes around it. Services do not apply. |
| `custom` | Whatever the rules say. Rules match traffic by domain, suffix, IP, CIDR, service or process and send it to a named egress route. Unmatched traffic uses the default route. |

## Two kinds of route

- A **kernel route** is an entry VPN Bypass has installed in the macOS routing table right now, one per resolved address or range: `list_active_routes`, `clear_routes`, `refresh_routes`.
- A **Custom-mode route** is a named egress that rules point at: the VPN (optionally one pinned tunnel), Direct, an HTTP or SOCKS5 proxy, or a Tailscale exit node. The `custom_*` tools manage these.

## Tools

Read tools change nothing and stay registered in read-only mode. The last column is the socket command each tool sends.

| Tool | Read | Needs 4.9.0 | Arguments | Command |
|---|---|---|---|---|
| `status` | yes | | | `status` |
| `get_logs` | yes | yes | `limit` (1-200, default 50), `level` | `logs` |
| `set_mode` | | | `mode`: `bypass`, `vpnOnly`, `custom` | `mode` |
| `list_domains` | yes | yes | `list`: `bypass` or `vpnOnly` (both when omitted) | `domain.list` |
| `add_domain` | | yes | `domain` (a host name; on `bypass` also a link, whose host is saved, from VPN Bypass 5.0; on `vpnOnly` also an IPv4 CIDR), `list` (default `bypass`) | `domain.add` |
| `remove_domain` | | yes | `id` or `domain`, `list` | `domain.rm` |
| `set_domain_enabled` | | yes | `id` or `domain`, `enabled`, `list` | `domain.enable`, `domain.disable` |
| `list_services` | yes | yes | | `service.list` |
| `get_service` | yes | yes | `id` | `service.list` |
| `set_service_enabled` | | yes | `id`, `enabled` | `service.enable`, `service.disable` |
| `list_active_routes` | yes | yes | `source` | `routes.active` |
| `clear_routes` | | yes | | `routes.clear` |
| `refresh_routes` | | yes | | `refresh` |
| `refresh_dns` | | yes | | `dns.refresh` |
| `custom_list_routes` | yes | | | `route.list` |
| `custom_add_route` | | | `name`, `type` (`http`, `socks5`, `tailscale`, `vpn`), `host`, `port`, `user`, `password`, `interface`, `product` | `route.add` |
| `custom_update_route` | | | `id`, then any of `name`, `host`, `port`, `user`, `enabled`, `password` (an empty `user` or `password` removes it) | `route.set` |
| `custom_set_route_enabled` | | | `id`, `enabled` | `route.enable`, `route.disable` |
| `custom_remove_route` | | | `id` | `route.rm` |
| `custom_list_rules` | yes | | | `rule.list` |
| `custom_add_rule` | | | `match`, `pattern`, `route_id` | `rule.add` |
| `custom_remove_rule` | | | `id` | `rule.rm` |
| `custom_set_default_route` | | | `route_id` | `default` |

Every tool carries the MCP hints `readOnlyHint`, `destructiveHint` and `idempotentHint`, so a client can ask before a destructive call. The destructive ones are `set_mode`, `remove_domain`, `clear_routes`, `custom_update_route`, `custom_remove_route`, `custom_remove_rule` and `custom_set_default_route`. `set_mode` is on the list because the app re-applies every kernel route on each call, even when the mode does not change.

## What happens after a change

The domain and service tools call the same code as the app's buttons. The app saves the list before it answers. It adds or removes the kernel routes for that one entry in the background, and only while a VPN is connected. `refresh_routes` and `refresh_dns` answer at once and do their work in the background. To see the effect, read `list_active_routes` or `get_logs` a few seconds later.

## Which tools need VPN Bypass 4.9.0

The domain, service, active-route, refresh and log tools use socket commands added in VPN Bypass 4.9.0. An older app answers them with:

```text
this needs VPN Bypass 4.9.0 or newer: the running app is older and does not know the domain.list command. Update VPN Bypass and try again.
```

## Errors

A failed call returns an MCP tool error with one line of text:

| Text | Meaning |
|---|---|
| `VPN Bypass is not running; start the app` | Nothing listens on the socket. |
| `this needs VPN Bypass 4.9.0 or newer...` | The app is older than the tool. |
| `invalid arguments: ...` | The server refused the arguments before sending anything, for example neither `id` nor `domain`, or a port of `80.5`. |
| `<code>: <message>` | The app's own error, passed through: `not_found`, `invalid_args`, `already_exists`, `invalid_port`, `invalid_type`, `helper_not_ready`, `timeout`, `request_too_large`. |

## Things to ask

- "Is VPN Bypass enforcing right now?"
- "Add example.com to the bypass list, then show me the routes it installed."
- "Which services are enabled?"
- "Show me the last 20 errors in the VPN Bypass log."
- "Switch my residential proxy route to port 24001."
