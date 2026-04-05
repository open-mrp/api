1. All PATCH and POST endpoints must respect idempotency keys sent by the user.
2. All PUT, DELETE, and GET endpoints must be designed to be idempotent by default without idempotency keys.
3. All microservice calls should be idempotent via internal idempotency keys even if the user does not supply them. 
4. All services inside microservices should change the database atomically wherever possible.
5. All business logic should be inside a service or mediator. See architecture-patterns.md for more information.
6. All apiresources should have an "Object" field.
7. Nested resources that are returned by the API should prefer json structures like so:

```json
{
    // ...
    "user": { "id": "us_...", "object": "user" },
    // ...
}
```

rather than:

```json
{
    // ...
    "user_id": "us_...",
    // ...
}
```

8. New endpoints should be added to the openapi spec generator.
9. Sensitive response or requests should be sanitized using the ShieldRequestBody or ShieldResponseBody APIEndpointExtras.