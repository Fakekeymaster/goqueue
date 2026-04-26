### What is Flags().IntVarP?
`IntVarP` registers a flag that:

- Stores into &flagWorkers (a pointer to our variable)
- Long form: --workers
- Short form: -w (the P in IntVarP = "with short flag")
- Default: 0
- Help text shown in --help


### Why map[string]any and not queue.Job?
The CLI is a separate binary that talks to the server over HTTP. It doesn't need to import internal packages — it just reads JSON. Using map[string]any keeps the CLI thin and decoupled. If you later rewrite the server in a different language, the CLI still works unchanged.

### What is any?
any is an alias for interface{} introduced in Go 1.18. It means "any type." Since JSON values can be strings, numbers, booleans, or null, we need a flexible container. map[string]any matches the structure of any JSON object.