# Capability Discovery

New task authoring must not infer engine availability from documentation or an
engine name. All endpoints below require `X-Internal-Token`.

`GET /api/v1/capabilities` returns
`validation-engine-capabilities.v1`: a sorted list of configured engine IDs,
supported validation-contract versions, allowed authoring modes, execution
class, accepted workspace input forms, and whether the engine is allowed for
new authoring. The envelope includes a SHA-256 digest of the exact list.
`legacy.generic` is discoverable for operations but is never authoring-ready.

`POST /api/v1/contracts/inspect` strictly parses a V1 contract without calling
an engine. It returns deterministic stage order, required and missing engines,
the normalized contract digest, the current capability digest, and `runnable`.
Inspection proves only that the contract can be planned with the currently
configured registry. It does not prove that a starter fails, a reference
passes, or a negative fixture is detected.

Runtime-suffixed engines are advertised final-only for new authoring. This is a
fail-closed authoring capability even where old validation requests can still
exercise broader compatibility behavior.
