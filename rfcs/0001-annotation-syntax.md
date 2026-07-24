# RFC 0001: Spice Annotation Syntax

Status: Proposed

## Summary

Use Go declaration comments in the canonical form `// @Name(arguments...)`.

## Goals

- Preserve valid Go source.
- Support familiar annotation ergonomics.
- Enable source-positioned diagnostics.
- Support built-in, qualified, and user-defined annotations.
- Avoid a compiler fork.

## Initial grammar

```text
annotation  = "@" name [ "(" [ arguments ] ")" ]
arguments   = argument { "," argument }
argument    = [ identifier "=" ] value
value       = string | integer | boolean | identifier | list
list        = "[" [ value { "," value } ] "]"
```

## Open questions

- Nested annotation values.
- Type references and symbol references.
- Alias and composed annotation behavior.
- IDE completion protocol.
