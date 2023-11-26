POST http://localhost:8080/shows
Correlation-Id: tacYQ42Lau2fJnN3v7dCxe

{
  "dead_nation_id": "8dd49e96-6872-4a95-bd8d-25b6580a78f7",
  "number_of_tickets": 100,
  "start_time": "2024-04-28T19:00:00Z",
  "title": "Evil Nëcrømancer: Raging Doom 2024 Tour",
  "venue": "The Catacombs"
}

HTTP 201

[Asserts]
jsonpath "$.show_id" exists

[Captures]
show_id: jsonpath "$.show_id"

POST http://localhost:8080/book-vip-bundle
Correlation-Id: yMGhzzZ7ywzxrHHYPfL65L

{
  "customer_email": "email@example.com",
  "inbound_flight_id": "d433534b-af78-4b94-9507-90ff3a40f1c5",
  "number_of_tickets": 4,
  "passengers": ["Jane Doe", "Jane Smith", "John Doe", "John Smith"],
  "return_flight_id": "d308acd4-fdb9-4d87-8bef-b88995ae07c3",
  "show_id": "{{show_id}}"
}

HTTP 201

POST http://localhost:8080/book-vip-bundle
Correlation-Id: yMGhzzZ7ywzxrHHYPfL65L

{
  "customer_email": "email@example.com",
  "inbound_flight_id": "d433534b-af78-4b94-9507-90ff3a40f1c5",
  "number_of_tickets": 100,
  "passengers": ["Jane Doe", "Jane Smith", "John Doe", "John Smith"],
  "return_flight_id": "d308acd4-fdb9-4d87-8bef-b88995ae07c3",
  "show_id": "{{show_id}}"
}

HTTP 201