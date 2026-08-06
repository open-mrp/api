# API Design Principles

> Source: https://www.danealbaugh.com/articles/api-design-principles

Four philosophies, not practices — they exist so you can make sound decisions
when no established practice covers the situation.

## 1. Make no assumptions

Never assume anything will work as expected or that users will behave
predictably.

- Network connections will fail.
- Operations expected to never fail will fail.
- Users will do unexpected things.
- Every observable behavior of the system becomes someone's dependency
  (Hyrum's Law in spirit) — so an "accidental" behavior is still a contract.

Design defensively; account for the failure scenario before the happy path.

## 2. Make things as simple as possible

Use the fewest concepts necessary to solve the problem.

> "Perfection is achieved not when there is nothing more to add, but when
> there is nothing more to take away." — Antoine de Saint-Exupéry

Ruthlessly eliminate unnecessary complexity and features.

## 3. Separate concerns

Organize code into distinct sections addressing single responsibilities,
using abstraction to hide implementation complexity behind clean interfaces.

Benefits:
- **Maintainability** — single-responsibility layers are easier to understand
  and modify.
- **Testability** — layers can be mocked because consumers depend on
  interfaces, not implementations.
- **Loose coupling** — a change inside one layer has minimal impact on
  others; layers evolve independently.

Symptoms of a poor abstraction:
- Hidden relationships and dependencies.
- Overly simplistic modeling.
- Unnecessary or restrictive coupling.

Layer the system with clean data models and interfaces exposed upward.

## 4. APIs are for humans

The developers using the API are human and deserve optimal usability.

- First designs are rarely optimal.
- Actively seek user feedback — don't wait for it passively.
- Observe actual usage patterns.
- Identify and eliminate friction points.
- Empathize with users; solve problems they haven't verbalized.

Prioritize developer experience through iterative, observation-based
refinement.
