# Executable Contract Verification

`POST /api/v1/contracts/verify` accepts
`validation-contract-verification-request.v1` and runs the supplied final V1
contract against isolated sandbox workspace references. Fixture source and
private reference contents are not uploaded to the orchestrator.

A request contains exact blueprint and runtime-profile SHA-256 digests and:

- exactly one `starter` case, which must fail validation;
- exactly one `reference` case, which must pass;
- one or more `negative` cases, each of which must fail.

Every case uses a distinct canonical `/workspaces/<sandbox-id>` root. The
orchestrator first performs strict inspection and rejects unavailable engines,
then executes cases sequentially in authored order. An outcome mismatch makes
the receipt non-passing.

The `validation-contract-verification-receipt.v1` response freezes the
blueprint, runtime-profile, normalized contract, and configured-capability
digests. It includes only expected/actual booleans and hashes of normalized
case results, so the receipt does not expose private fixtures or raw engine
evidence. Its own digest covers the complete receipt before the digest field is
set.

A receipt is evidence from the validation service, not publication authority.
The practice-task owner must persist it against the exact immutable revision,
reject digest drift, and require `passed=true` before moving a revision to
`RUNNABLE` or `PUBLISHED`.
