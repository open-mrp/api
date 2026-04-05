1. Ensure the apiresource looks okay
- Constants where appropriate & match legacy enums
- Expandable noted where appropriate
- Good comments for generated documentation
- No e.g. where constants should be
- enum=<resource_type> for the object type
- No omitempty when appropriate
- Consistent naming conventions

2. Ensure the request data looks okay
- Constants where appropriate
- Include params where appropriate
- Good comments for generated documentation
- No e.g. where constants should be

3. Double check endpoint definition
- Make sure it should be public
- Double check title & description is appropriate
- Make sure domain makes sense

4. Double check presenter layer and service in api-gateway
- Get rid of unnecessary helper functions
- Make sure all presenter items are set

5. Check grpc handler
- If idempotent, make sure key is handled from context
- 

6. Check service handler
- Identity checks in place with permissions as appropriate
- Make sure the new identity patterns are used
- Proper TX management
- If idempotent, proper recovery point management
- Ensure audit events are included
- Ensure delete resource records are created if needed

7. Mediator checks
- Make sure discrete business logic is handled in a mediator rather than a service

6. Repository checks
- Make sure deletes insert to deleted resource table
- Make sure deleted records do not orphan records
- Make sure all error messages are human readable
