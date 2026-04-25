These directives define the core philosophy of this project. As an agent, these are the immutable laws that govern every code contribution and architectural decision.

## 1. Intent vs. Execution
*   **Directive**: Never conflate *what* the system does with *how* it is powered.
*   **Abstraction**: The core logic (Intent) must exist in a vacuum, oblivious to the external world. Infrastructure (Execution) is a pluggable detail that serves the Intent, never the other way around.

## 2. Sovereignty of the Core
*   **Directive**: Protect the inner logic from external corruption.
*   **Abstraction**: The domain must be self-sufficient. It defines its own needs through contracts. External libraries, transport layers, and databases are "foreign powers" that must adapt to the core's protocols.

## 3. Explicit Lineage
*   **Directive**: Dependencies must be declared at the gates, never hidden in the shadows.
*   **Abstraction**: All connections must be visible and passed through the front door. Global state and hidden side-effects are prohibited. The assembly of the system happens in a single, observable location.

## 4. Radical Minimalism
*   **Directive**: Value the built-in over the borrowed.
*   **Abstraction**: Complexity is a debt that must be justified. Every external dependency is a loss of sovereignty. Prioritize the fundamental primitives of the language to maintain a lean, understandable, and long-lived system.

## 5. Universal Traceability
*   **Directive**: Every action must leave a traceable path.
*   **Abstraction**: A system that cannot be observed is a system that cannot be trusted. Propagation of identity across boundaries is mandatory. Failure must be transparent, carrying its own context from the point of origin to the boundary.

## 6. Forward-Leaning Standards
*   **Directive**: Align with the future, not the past.
*   **Abstraction**: Adopt standards that favor predictability and modern efficiency. Choose patterns that are time-ordered and deterministic, ensuring the system remains relevant and performant as it scales.

---
*These directives are the foundation of Go-Microne. Adhere to them to ensure the system remains elegant, decoupled, and immortal.*
