# Restaurant Mock

`restaurant-mock` имитирует внешнюю систему ресторана и позволяет проверить интеграцию без реального партнёра.

## API

- `GET /health` — проверка доступности.
- `GET /api/v1/menu` — демонстрационное меню.
- `POST /api/v1/orders` — создание заказа.
- `GET /api/v1/orders/{id}` — получение заказа.
- `PATCH /api/v1/orders/{id}/status` — изменение статуса.

## Статусы

```text
accepted -> preparing -> ready -> delivered
    |          |          |
    +----------+----------+----> cancelled
```

Допускаются повторная установка текущего статуса и переходы:

- `accepted` -> `preparing`, `cancelled`;
- `preparing` -> `ready`, `cancelled`;
- `ready` -> `delivered`, `cancelled`.

Из `delivered` и `cancelled` переходы запрещены.

## Idempotency

Mock использует `order_id` из Kitchen API как ключ идемпотентности. Если Kitchen повторяет запрос после `5xx` или сетевой ошибки, mock возвращает уже созданный ресторанный заказ вместо создания дубликата.

## Failure injection

Переменные Docker Compose:

- `MOCK_FAIL_COUNT=2` — первые два запроса создания заказа завершаются ошибкой.
- `MOCK_FAIL_STATUS=503` — HTTP-код временной ошибки.
- `MOCK_DELAY_MS=3000` — искусственная задержка ответа.

Это позволяет вручную проверить retry и timeout.
