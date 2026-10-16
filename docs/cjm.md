# Customer journey

1. Customer opens the service.
2. Customer selects a restaurant.
3. Customer views the menu.
4. Customer adds dishes to a cart.
5. Customer reviews the total.
6. Customer submits an order.
7. Kitchen API creates a pending order and sends it to the restaurant.
8. Restaurant accepts or rejects the order.
9. Customer receives the current order state.

# Restaurant journey

1. Restaurant receives an order from Kitchen API.
2. Restaurant validates the order.
3. Restaurant accepts/rejects it.
4. Restaurant can continue processing it independently.
5. Kitchen API stores the external order identifier and status.
