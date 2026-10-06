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

For retained consumers that require Files, production reads each case once through the domain
`WorkspaceFileReader` port and its `internal/infrastructure/workspace` adapter.
`VERIFICATION_WORKSPACES_DIR` maps the portable ID to a trusted absolute,
canonical physical mount; its default is `/workspaces`. Engines receive both
the original portable root and sorted relative `{path,content}` files. The
reader uses Go 1.25 `os.Root` descriptors for the base, case and descendants,
rejects symlink components and special files, and checks cancellation throughout.
Native macOS/Linux file opens use a nonblocking flag to avoid a FIFO hang.
Explicit pre/post-open Lstat type and descriptor identity checks reject an
observed symlink or path replacement before reading. `os.Root` itself permits
contained symlinks, even with a caller no-follow flag, while preserving the
descriptor containment boundary. These observed guards require the declared
read-only trusted fixture mount; they do not promise an atomic filesystem
snapshot or impossibility of arbitrary concurrent mutation.

The supported projection has 1–1000 regular text files, at most 4 MiB per file
and 16 MiB total, enforced on actual bounded reads. Contents and paths must be
UTF-8. The existing content string permits U+0000 and those bytes are preserved
for the language engine to judge; NUL remains forbidden in file paths. Walks are technically bounded
to 10,000 entries and 128 nested directories, including empty directories, so
an empty-directory tree cannot consume unbounded resources. These guards do
not represent directory, mode, Git/history or environment semantics. Missing
reader/mount, unsafe or unsupported files, cancellation and engine execution
errors fail technically without an executable receipt; they never count as
expected starter/negative semantic failures. Error messages expose neither
physical paths nor private fixture contents.

Only authoring VerifyContract performs this hydration. Ordinary V1 Execute
and official Practice V2 pinned-snapshot reads are unchanged; there is no local
live-head fallback for official validation.

Hydration is conditional on the same FINAL stage/link filtering used by
execution. Actual configured adapters declare their input requirement via
the optional `NeedsWorkspaceFiles(stage)` domain interface. HTML/CSS/SCSS,
React, PHP and Node require Files. Foundation declarations follow existing
provider DTOs: code/PHP-framework/Python/NextJS/Browser/Docker/database-static
consume Files; SQL runtime setupFiles requires Files only when authored;
cache/search static explicit targets can use RootPath fallback. Git, Linux and
dedicated HTTP/framework runtime keep root dispatch and their existing provider
prerequisites. HTTP dispatch shares the same selected-client branch for both
execution and declaration. Retained `workspace.file_contains` and
`workspace.selector_exists` also need Files. Filtered-out consumers add no
mount dependency. The public capability listing is not used to infer support;
unclassified custom clients keep prior dispatch without a repaired-files claim.

A mixed root/text contract still needs the complete supported text projection.
Unsupported encoding or rich artifacts fail technical; the reader never skips
Git metadata or reconstructs it from JSON files. Root-only authoring does not
read the local tree, and is unaffected by absent/unavailable Orchestrator mounts.

The checked-in Orchestrator Compose configuration has no workspace volume.
Before using file-consuming verification in deployment, provide a matching read-only mount of
the Sandbox-created case trees at the configured physical root and verify that
mapping. A configured default path alone is insufficient. An isolated test
overlay proves the scoped reader/provider composition, not production mount
readiness or a sealed immutable attestation.

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
