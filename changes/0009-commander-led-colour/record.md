---
kind: feature
status: draft
timestamp: 2026-11-15T00:48:51Z
---

# A static colour for the LEDs of the Commander Core XT in run

## Log

- 2026-11-15T00:48:51Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, wrote the spec from the facts the
  orchestrating session gave of the trace of OpenLinkHub and of its tests on the device with the human watching the
  LEDs, from the trace itself, read for the traffic alone, which added the endpoint byte at the end of a read and a
  close, the close before each open on channel 1, the answer of `0x1A`, the command `08 02 13` and the write of zero
  triples before a static colour, from the decisions of the orchestrating session on the key, the session, the
  keepalive, the failures and the documents, and from the specs of units 0004 to 0008, the controller document and the
  driver, the config and the loop of `run` in the tree. No code of OpenLinkHub, liquidctl or OpenRGB was read. The
  status is `draft`
