# Task: Implement Share Rules and Restore Features

## TS: 2025-03-23 15:54:58 CET

## PROBLEM:

Users need a way to share their Cursor Rules configurations with others, while ensuring privacy and handling conflicts appropriately. The current system doesn't provide a way to export rules in a shareable format or import them from another user's shared configuration.

## WHAT WAS DONE:

- Created task plan for implementing "share rules" and "restore from shared" features
- Outlined design for creating shareable rule exports that strip personal data
- Designed approach for conflict resolution during rule restoration
- Planned support for embedded .mdc content for local rules sharing

## MEMO:

This feature will enable collaboration through rule sharing, similar to package.json/requirements.txt in other package managers. The implementation distinguishes between local lockfiles (with personal paths) and shareable exports (sanitized for sharing). Future enhancements could include support for private repositories and a centralized registry.

## Task Steps

1. [ ] Design Shareable File Format

   - [ ] Define JSON schema with formatVersion field
   - [ ] Determine which fields to include/exclude from RuleSource
   - [ ] Add fields for embedded .mdc content (optional)
   - [ ] Create structs for serialization/deserialization

2. [ ] Implement "Share Rules" Command

   - [ ] Load local lockfile into memory
   - [ ] Filter and sanitize entries (remove personal paths)
   - [ ] Handle local references (mark as unshareable or embed content)
   - [ ] Write sanitized JSON to shareable file
   - [ ] Add CLI command `cursor-rules share [--output <file>]`

3. [ ] Implement "Restore From Shared" Command

   - [ ] Parse and validate shared file format
   - [ ] Implement conflict resolution strategies
   - [ ] Handle different sourceTypes (github-file, embedded, etc.)
   - [ ] Download/extract required .mdc files
   - [ ] Update local lockfile with restored rules
   - [ ] Add CLI command `cursor-rules restore <file>`

4. [ ] Add Privacy Protections

   - [ ] Ensure no personal paths/tokens are included in shares
   - [ ] Add warnings about sharing private repository references
   - [ ] Implement sanitization for potentially sensitive data

5. [ ] Implement Conflict Resolution

   - [ ] Detect existing rules with same key during restore
   - [ ] Add options for overwrite/rename/skip
   - [ ] Implement auto-rename functionality (e.g., task-it-2)
   - [ ] Update lockfile with conflict resolution decisions

6. [ ] Add Support for Embedded .mdc Content

   - [ ] Implement content encoding/decoding in shareable file
   - [ ] Extract and save embedded .mdc during restore
   - [ ] Add options to include/exclude content during share

7. [ ] Update Documentation and Examples

   - [ ] Document share/restore commands with examples
   - [ ] Explain shareable file format and limitations
   - [ ] Provide guidance on sharing best practices
   - [ ] Update README.md with new functionality

8. [ ] Add Tests
   - [ ] Test sharing rules with various configurations
   - [ ] Test restoring from shared files with conflicts
   - [ ] Test privacy protections and sanitization
   - [ ] Test embedded content functionality
