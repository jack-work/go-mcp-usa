
# TODO:
- Figure out compatibility between model types.  I wrote some wrapper types meant to abstract generic LLM API data that end up conflicting with the jsonrpc types.  I don't know the best Go patterns but I think I can do better than manual conversion.  Should be able to type check and cast, or creatively use embedding
- For now it works.

- Immediate next steps are to handle tool use and implement a loop in the one shot message

> [!IMPORTANT]
> Implement decorator pattern for logging
