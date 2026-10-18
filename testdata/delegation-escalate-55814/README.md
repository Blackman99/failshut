Modeled on anomalyco/opencode#55814. `plan` keeps its shipped edit deny but
does not deny `subagent`, so it can delegate to `general`, which writes files.
`reviewer` denies edit, but a later `subagent * allow` wins (last match).
