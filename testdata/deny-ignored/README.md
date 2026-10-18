The exact config from anomalyco/opencode#55728. The leading `./` never matches
the project-relative path the read tool checks, so the deny has no effect.
