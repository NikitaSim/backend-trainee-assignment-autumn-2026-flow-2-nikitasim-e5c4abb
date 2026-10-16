# Architecture

```text
Client
  |
  v
kitchen-api
  |-- handler
  |-- service
  |-- repository ---> PostgreSQL
  |
  `-- restaurant client ---> restaurant-mock
```

The MVP deliberately uses two deployable applications rather than a large number of microservices.

## Order flow

1. Read cart.
2. Validate that it is not empty.
3. Persist a pending order.
4. Send the order to the restaurant.
5. Save the external order ID and mark the order confirmed.
6. If the restaurant is unavailable, keep the order pending/failed according to the final business rule.

For production, the integration can be moved to an outbox + message broker without changing the public API.
