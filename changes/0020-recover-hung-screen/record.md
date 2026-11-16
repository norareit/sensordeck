---
kind: bug
status: implemented
timestamp: 2026-11-16T13:25:49Z
supersedes: 0004-linux-sensors-and-run
---

# Recovering a screen whose firmware hung after a USB reset

## Log

- 2026-11-16T13:25:49Z [Claude Code, anthropic/claude-opus-5-5] as the specifier, wrote the spec from the report of
  the orchestrating session of a screen that shows no values after a cold boot, and the three facts it established on
  the device on 2026-11-16, against the udev rule, the Linux transport and its discovery, the feeder of `run` and its
  theme holder, the upload, the one-shot commands, the fake screen and `internal/fakepty`, `docs/protocol.md`, the
  README and the specs of units 0004, 0013 and 0017. Read on the human's machine: the screen's USB device has
  `bmAttributes` `0x80` and no `power/wakeup`, so it cannot wake the host and usbhid keeps it resumed while its node is
  open. Decided: the rule sets `autosuspend_delay_ms` to 2000 and `control` to `auto` on the USB device, with no
  `ACTION` match; the first timed-out write is a hang; `run` recovers at the next round by closing the node, waiting
  for `suspended` up to the delay plus 5 seconds, and reopening; the lines of a hang are written once each, and
  `screen recovered` at the first write that succeeds; without the rule, one line of what to do; the one-shot commands
  report and do not recover; no end-to-end test, because no write to the fake screen's terminal can time out. The
  status is `draft`
- 2026-11-16T13:28:21Z [Claude Code, anthropic/claude-opus-5-5] read the spec under the delegation in `AGENTS.md`: its facts are those
  the orchestrating session and the human observed on the device on 2026-11-16, what was observed once or inferred is
  marked as such, sensordeck sends no USB reset and writes no sysfs, and every requirement can be tested in the
  sandbox with a temporary sysfs tree and a fake transport. That opening the node resumes the device, and that a
  healthy screen suspends harmlessly, go to the human's check after the next cold boot; the unit does not depend on it.
  Item 3 of "Beyond the request" is not taken into the unit: the orchestrating session adds the amendment to the record
  of `0004-linux-sensors-and-run` itself when it merges this unit. Items 1 and 2 are declined. The approval that
  follows is made under that delegation
- 2026-11-16T13:28:22Z [human] approved the spec. The status is `approved`
- 2026-11-16T13:31:57Z [Claude Code, anthropic/claude-opus-5-5] as the planner, wrote `plan.md` against the tree of the
  branch: five commits, for the udev rule, the reading of the USB power state in the Linux transport, the seam of the
  command line with the one-shot commands, the recovery of `run`, and the documents, with each requirement traced to
  its tasks. The status stays `approved`
- 2026-11-16T13:32:28Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, began the work. The status is
  `in-progress`
- 2026-11-16T13:37:08Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, built the five tasks of `plan.md` in
  five commits and ticked each as it was done. The `screenPower` seam stands as planned. The theme holder needed no
  handing of the wrapper, since `ensure` takes the transport at each call and the feeder passes its observed wrapper.
  The tests of `run` use `linuxPower` on a `fakeusb` tree, as planned. The tests of the hang were written after the code
  of the same commit and not before it, so none of them was seen to fail first. Decided during the work: the feeder
  reads `suspended` before the first wait, so that a device already suspended is not waited for. On `7bdfb2e`, the five
  commands of the suite were run and reported nothing but passes: `go test ./...` passed, `go vet ./...`, `gofmt -l .`,
  and the Windows build and vet printed nothing. Not checked: the behaviour on the device, which is the human's under
  D13 of the spec, and the claim of the spec that an open node keeps the device resumed. The status stays `in-progress`
- 2026-11-16T13:39:09Z [OpenCode, openai/gpt-6.1-sol] as the reviewer, made a full review of unit
  `0020-recover-hung-screen` at `25a80dad65ccebabc0aaabe3dda9e80ee2c1b133`, against the merge base with
  `origin/master`. Returned P1 in `.reviews/0020-recover-hung-screen/review-round-1.md`. Ran all five required
  checks, a fresh `go test -count=1 ./...` including end-to-end tests, `npm run check-changes` and
  `git diff --check`; all passed. Could not verify the physical screen's recovery, resume on opening or healthy
  operation with autosuspend enabled, or Windows runtime behavior. The status stays `in-progress`
- 2026-11-16T13:40:34Z [Claude Code, anthropic/claude-sonnet-5-5] as the implementer, answered finding P1 of
  `.reviews/0020-recover-hung-screen/review-round-1.md`. Accepted by the definition of the orchestrator, for the part
  that is mine to apply: the tests of the hang were validated retrospectively. With `failed` of the feeder made to
  ignore a timeout, so that a timed-out write is the screen lost as before the unit, 11 tests of `run` failed
  (`TestRunRecoversAHungScreenBySuspendingAndReopeningIt`, `TestRunChecksTheThemeAfterTheRecoveryOfAHungScreen`,
  `TestRunOpensTheNodeAfterTheLimitWhenTheDeviceNeverSuspends`, `TestRunWritesTheLinesOfARecoveryThatDoesNotHelpOnce`,
  `TestRunSaysThatTheDeviceDidNotSuspendOnceOverTwoRecoveries`, `TestRunReopensAHungScreenPlainlyWhenNoRecoveryIsPossible`,
  `TestRunReadsThePowerFilesAtEachRoundOfAHang`, `TestRunWritesThatAnUploadFailedBeforeThatTheScreenHung`,
  `TestRunStopsDuringTheWaitForTheSuspend`, `TestRunEndsAHangWithoutALineWhenAnotherFailureFollows`,
  `TestRunWritesTheLinesOfASecondHangAgain`), and the change was then reverted, so the tree has the final code. The
  tests of `hangError` and of `hidraw` were not validated this way. The plan task is annotated as written after the
  code. This is retrospective validation and not test-first execution. Proposed decline, for the human: the rule
  that the test comes first was broken, and the exception the review asks for is the human's to record, so none is
  recorded here. The status stays `in-progress`
- 2026-11-16T13:40:51Z [Claude Code, anthropic/claude-opus-5-5] under the delegation in `AGENTS.md`, decided the proposed decline of
  P1 of round 1: declined, and an exception to the order of tests first is recorded for the tests of the hang in
  `run`. The order cannot be restored after the fact, the record states it, and the implementer showed that 11 of
  the tests fail with the classification of timeouts disabled, so they guard the fix. P1 does not block
- 2026-11-16T13:41:32Z [OpenCode, openai/gpt-6.1-sol] as the reviewer, made a delta review of unit
  `0020-recover-hung-screen` at `c3bf07df6cad6209504427ce5dda037194eda1ae` from
  `25a80dad65ccebabc0aaabe3dda9e80ee2c1b133`, saved in
  `.reviews/0020-recover-hung-screen/review-round-2.md`. Checked the response to P1 and returned no new findings.
  Ran fresh `go test -count=1 ./...` including end-to-end tests, both platform vet commands, the Windows build,
  `gofmt -l .`, `npm run check-changes` and `git diff --check`; all passed. Could not independently verify the
  historical mutation experiment, physical-device behavior or Windows runtime behavior. The status stays `in-progress`
- 2026-11-16T13:43:49Z [Claude Code, anthropic/claude-opus-5-5] under the delegation in `AGENTS.md`, decided the findings: P1 of
  round 1 was corrected in the documents with its exception declined as above, and round 2 returned none, so no
  blocking finding is open. The timestamps of three entries of the implementer, which did not match their commits,
  were corrected to the times of the commits that added them. Every task of `plan.md` is ticked, and the five
  commands of the suite passed on the branch. The recovery at a cold boot with the new udev rule is the human's check
  and follows the merge. The status is `implemented`
