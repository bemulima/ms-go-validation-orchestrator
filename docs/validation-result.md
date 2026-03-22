# Validation Result V1

`ValidationResultV1` is the canonical normalized output returned by `ms-go-validation-orchestrator`.

## Root fields

- `contract_kind`: source contract kind such as `workspace_contract` or `legacy_contract`.
- `contract_version`: contract version.
- `legacy`: whether the request came through legacy adaptation.
- `passed`: final aggregated pass/fail status.
- `stages[]`: stage-level reports.
- `links[]`: link-level reports.
- `errors[]`: top-level non-stage issues such as orchestration or compatibility errors.

## Stage report

- `stage_id`: stable stage identifier from the contract.
- `engine`: engine ID such as `html.dom` or `node.express`.
- `status`: orchestration status such as `passed`, `failed`, `skipped`, `blocked`.
- `passed`: boolean pass/fail flag.
- `optional`: whether the stage is non-blocking.
- `duration_ms`: measured stage duration.
- `evidence[]`: normalized proof points that can be rendered in UI.
- `errors[]`: normalized validation issues.
- `warnings[]`: non-fatal issues returned by the engine.

## Link report

- `link_id`: stable link identifier from the contract.
- `kind`: link kind such as `workspace.file_contains`.
- `status`: link execution status.
- `passed`: boolean pass/fail flag.
- `optional`: whether the link is non-blocking.
- `errors[]`: normalized link errors.

## Validation issue

- `code`: machine-readable issue code.
- `message`: human-readable message.
- `severity`: `error`, `warning`, or engine-specific normalized level.
- `stage_id`: origin stage when applicable.
- `engine`: origin engine when applicable.
- `file`: related workspace file.
- `selector`: related DOM/CSS selector.
- `route`: related HTTP or framework route.
- `symbol`: related language symbol such as a function, class, variable, or import.
- `property`: related CSS/property-level field.
- `hint`: student-facing or author-facing remediation hint.
- `line`, `column`: source location when available.

## Validation point

Evidence points are intentionally lighter than errors. They exist to support:

- student UI summaries
- stage detail views
- future deep-linking from issue to file/section
- cross-stage reasoning

Fields:

- `file`
- `selector`
- `route`
- `symbol`
- `property`
- `message`

## Example

```json
{
  "contract_kind": "workspace_contract",
  "contract_version": 1,
  "legacy": false,
  "passed": false,
  "stages": [
    {
      "stage_id": "css",
      "engine": "css.ast",
      "status": "failed",
      "passed": false,
      "optional": false,
      "duration_ms": 18,
      "errors": [
        {
          "code": "DECLARATION_VALUE_MISMATCH",
          "message": "Expected .card to define background-color.",
          "severity": "error",
          "stage_id": "css",
          "engine": "css.ast",
          "file": "styles.css",
          "selector": ".card",
          "property": "background-color",
          "hint": "Add background-color to .card."
        }
      ]
    }
  ],
  "links": [],
  "errors": []
}
```

## UI guidance

Student UI should group this payload by:

- stage
- file
- severity
- selector / route / symbol / property

Admin UI and debugging tools should preserve:

- raw stage order
- optional vs blocking stages
- top-level orchestration errors
