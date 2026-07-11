# cloud-dice-tray

Web-based dice roller for tabletop role-playing games.

Project planning and agreed technical decisions are recorded in:

- [`docs/architecture.md`](docs/architecture.md)
- [`docs/roadmap.md`](docs/roadmap.md)

## Dice expression package

The first implementation increment is the reusable parser and evaluator in
`internal/dice`. It supports polyhedral and Fudge dice, arithmetic, flattened
lists, aggregation/filter functions, rounding, structured errors, and a raw
roll trace.

```go
expression, err := dice.Parse("sum(maxk(3, 4d6))") // validates without rolling
result, err := expression.Evaluate()               // rolls exactly once
```

## Run locally

Copy the example YAML configuration, then start the embedded web workbench:

```bash
cp config.example.yaml config.yaml
go run . -config config.yaml
```

Open <http://127.0.0.1:8080>. The workbench has separate server-side
validation and roll actions, so validation never consumes randomness.

## Test

```bash
go test ./...
```
