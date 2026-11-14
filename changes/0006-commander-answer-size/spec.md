# The size of an answer of the Commander Core XT

## Context

Unit `0005-commander-core-xt-probes` added the temperature probes of the Corsair iCUE Commander Core XT as the metrics
`commander.<n>.temp`. On the human's real controller, `sensordeck sensors` lists none of them. This unit corrects that
defect of unit 0005. The licensing rule of unit 0005 still holds: no code of liquidctl is read, translated or copied,
and the facts below come from a trace of sensordeck itself and from the sysfs of the host.

"R*n* of unit 0005" names a requirement of the spec of that unit, and "the controller", "a reading of the controller"
and the steps 1 to 8 mean what they mean there. Every requirement of unit 0005 that this spec does not change still
holds.

### Observed behavior

An `strace` of `sensordeck sensors` on the host, with the controller connected, shows that sensordeck:

- opens `/dev/hidraw9`, the node of interface 0, which is the right node by R3 of unit 0005
- writes step 1 as 97 bytes, as R5 of unit 0005 says
- reads its answer into a buffer of 97 bytes, and the read gives 97 bytes, starting with
  `00 01 00 10 00 02 00 75 01 00 E3 00`: the echo `01` of step 1 in byte 1, `00` in byte 2, and stale temperatures
  after them, as unit 0005 describes
- writes step 8 at once, reads its answer, and closes the node

No line of the controller is printed. The answer of step 1 is a good answer, and R6 of unit 0005 rejects it only
because it requires that exactly 96 bytes were read.

### Expected behavior

The answer of step 1 above is valid, the sequence goes on to step 8 as R5 of unit 0005 describes, and `sensors` prints
`commander.1.temp` and `commander.2.temp` with the temperatures of the two probes.

### Steps to reproduce

1. Connect a Commander Core XT with two probes, and give the user access to its nodes
2. Run `sensordeck sensors`
3. No line starting with `commander.` is printed. Under `strace -e trace=openat,read,write`, the first read on the
   node of interface 0 returns 97, and the next write is that of step 8

The defect was reproduced by the human on the real controller, and the trace above is from that run.

### Root cause

The HID report descriptor of interface 0, read from `/sys/class/hidraw/hidraw9/device/report_descriptor` on the host,
is

```
06 42 FF 09 01 A1 01 09 01 15 80 25 7F 75 08 96 80 01 81 02 09 01 15 80 25 7F 75 08 96 80 01 91 02 C0
```

It names the vendor usage page `0xFF42`, declares no Report ID item, and declares one input report and one output
report, each of 384 (`0x180`) bytes of 8 bits. A read of a hidraw node gives at most the size of its buffer of one
report and discards the rest of that report. The 96 bytes of the trace of unit 0005 came from the other tool reading
into a buffer of 96 bytes, and the 97 bytes above come from sensordeck reading into a buffer of 97. Both are
truncations of the same report of 384 bytes. Unit 0005 took the size of the other tool's buffer for the size of the
answer and made it a condition of R6. Its fake controller answered with exactly 96 bytes, so its tests could not see
the difference.

Because the device has no report IDs, byte 0 of a read is the first data byte of the report, so the offsets of unit
0005 are right: the echo in byte 1, the status in byte 2, the data type in bytes 3 and 4, the count in byte 5 and the
probes from byte 6, as the 97 bytes above confirm. The leading `0x00` of a write is the report ID 0, which hidraw takes
off for a device without report IDs before the 96 data bytes go to the device. The writes of R5 of unit 0005 match
what the other tool sent and work on the device, as the echo of step 1 above shows, and this unit does not change them.

### Fix

Each answer is read into a buffer of the 384 bytes of the input report, an answer is valid when at least 96 bytes were
read, and only its first 96 bytes are interpreted. The fake controller answers with the full report of 384 bytes, so
that the tests read what the device gives.

## Requirements

### Reading an answer

- **R1** Each answer of steps 1 to 8 is read in one read into a buffer of 384 bytes, the size of the input report
  of the descriptor. Nothing else of R5 of unit 0005 changes: the writes stay 97 bytes, the eight commands, their
  order and the time limit stay as they are, and one read follows each write
- **R2** R6 of unit 0005 changes in its first condition only: an answer is valid when at least 96 bytes were read, its
  byte 1 is the second written byte of its command, and its byte 2 is `0x00`. For the answer of step 6, the data type
  in bytes 3 and 4 must be `10 00` and the count *c* in byte 5 must satisfy 6 + 3·*c* ≤ 96, as before. A read of fewer
  than 96 bytes is not valid. The rest of R6 of unit 0005, what ends the sequence and what is unavailable then, is
  unchanged
- **R3** Only bytes 0 to 95 of an answer are interpreted. What bytes 96 and later hold changes nothing of whether the
  answer is valid or of the readings that R9 of unit 0005 gives from it

### The fake controller

- **R4** The fake controller of R17 of unit 0005 answers each command on the node of interface 0 with the 384 bytes
  of a full input report, in one write to its side of the pseudo terminal: bytes 0 to 95 are the answer it gives today,
  failures included, and bytes 96 to 383 are `0xFF`. A command that it gives no answer to stays without one. Nothing
  else of R17 of unit 0005 changes
- **R5** A test of the fake controller reads the answer of a command into a buffer of 384 bytes and gets all 384 bytes
  in one read, with the first 96 as the answer of unit 0005 and the rest `0xFF`

### Tests

- **R6** Unit tests of the reading of an answer cover: an answer of 384 bytes, one of 97 bytes and one of exactly 96
  bytes, each with a good echo and status, are valid; one of 95 bytes is not; and the answer of step 6 of "What was
  observed" in unit 0005 as 384 bytes, with bytes 96 to 383 not zero, gives 37.3 and 22.7. The cases of the unit tests
  of unit 0005 that are about the size of an answer are changed to these, and the others stay
- **R7** A regression test of the source against the fake controller of R4 reads `commander.1.temp` 37.3 and
  `commander.2.temp` 22.7. It fails on the driver of unit 0005, which read into 97 bytes and required exactly 96, and
  the implementer records in `record.md` that it was seen to fail before the fix and pass after it
- **R8** The end-to-end tests of R20, R21 and R23 of unit 0005 run against the fake controller of R4 and keep their
  expected results, and every other test of the suite passes unchanged in what it expects

### The documents

- **R9** The section "Reports" of `docs/commander-core-xt.md` states, in the project's own words: the report
  descriptor of interface 0 and where on the host it was read; that it declares no report IDs and one input and one
  output report of 384 bytes each; that a read of a hidraw node gives at most the size of its buffer of one report and
  discards the rest of it, so that the 96 bytes of the trace were a truncation by the other tool's buffer; that without
  report IDs byte 0 of a read is the first byte of the report; that the leading `0x00` of a write is the report ID 0,
  which hidraw takes off, so that 96 bytes go to the device; and that sensordeck reads 384 bytes and interprets the
  first 96. The section "Unknown" names what bytes 96 to 383 of an answer hold and whether the device needs the full
  output report of 384 bytes, which it was not given and did not need. Every other statement of the document that
  says a read gives 96 bytes is corrected to match
- **R10** The `record.md` of unit 0005 has one more entry under "Amendments", made on this unit's branch and in the
  format of `process.md`, that names `0006-commander-answer-size` as the correction of the 96-byte condition of R6 and
  of the report size in `docs/commander-core-xt.md`, and says that the 96 bytes of its trace were the other tool's
  buffer. The `spec.md` and `plan.md` of unit 0005 do not change, and nothing else of its `record.md` changes. Taking
  this into the unit was item 1 of "Beyond the request", decided by the orchestrating session under the delegation in
  `AGENTS.md`

## Decisions

- **D1** The buffer of a read is 384 bytes, the size of the input report that the descriptor declares. A read then
  takes the whole report and hidraw discards nothing, so the driver sees what the device sent, and a later unit that
  needs bytes past 96 has them. A buffer of 96 or 97 bytes, as the other tool and unit 0005 used, was rejected: it
  works on hidraw only because hidraw discards the rest of a report, so it ties the driver to a truncation it cannot
  see, and on the pseudo terminal of the fake controller, which keeps the rest of a write for the next read, it would
  read the end of one answer as the start of the next. The cost is 288 bytes more per read, which is nothing
- **D2** The condition is at least 96 bytes, and not exactly 384. 96 bytes is all that the driver interprets and all
  that the trace of unit 0005 showed working. Requiring 384 would add a condition that no observation needs, and
  would fail on a device or a kernel that gives a shorter report for the same data, for no gain in telling a good
  answer from a bad one: the echo, the status and the data type do that, as D4 of unit 0005 says. The count stays
  bounded by the 96 bytes, as R2 says, because only those bytes are interpreted and nothing is known of the rest
- **D3** The fake controller writes the full report of 384 bytes, and does not imitate the truncation of hidraw. A
  pseudo terminal is a stream of bytes: the fake cannot see the size of the reader's buffer, and the bytes that a
  smaller buffer leaves stay for the next read rather than being discarded. A fake that writes exactly the report a
  real device sends is the closest it can come, and with the buffer of D1 the driver reads it as it reads the device.
  The limitation is that a test which reads into a buffer smaller than 384 bytes sees a stream that hidraw would not
  give; such a reader is a test of the fake, R5, and reads 384 bytes. A run on the host of 20000 writes of 384 bytes
  to the master of a raw pseudo terminal, each read by a reader already waiting with a buffer of 384 bytes, gave each
  write in one read, so the answers do not split in the tests
- **D4** The bytes 96 to 383 of the fake's answer are `0xFF` and not `0x00`. Their content on the device is unknown,
  and a value that is not zero lets a test show that they are not interpreted, which zeros, the default of most
  buffers, would hide
- **D5** The writes stay as R5 of unit 0005 says, 97 bytes with the report ID `0x00`. The device answers them, as the
  trace above shows, and the other tool wrote the same. Writing the full output report of 384 bytes would change
  traffic that works for no observed reason, and is named as unknown in the document instead
- **D6** The record sets `supersedes` to `0005-commander-core-xt-probes`. Unit 0005 is implemented and frozen, and
  its spec states as an observation and in R6 that an answer is 96 bytes, which this unit shows to be the other tool's
  buffer. `process.md` makes a correction that would leave the original record misleading a new unit that names the
  old one in `supersedes`. Only R1 and R2 of this unit replace a part of unit 0005, the size of the read and the first
  condition of R6, and the rest of unit 0005 stands
- **D7** The unit needs no `plan.md`. The fix is one commit: the size of the buffer and the condition of validity in
  the driver, the size of the answer in the fake controller, the tests of R5 to R7, the document, and the amendment
  of R10. The driver and the fake change together: the fake of R4 alone breaks the tests of unit 0005, and the driver
  alone passes against the fake of unit 0005, so that no test would show the defect it corrects

## Non-goals

- Any change to the commands, their order, the time limit, the discovery of the node or the parsing of the probes,
  beyond the size and the condition of R1 to R3
- Interpreting bytes 96 to 383 of an answer, or raising the bound of 30 probes that the count has within 96 bytes
- Writing the full output report of 384 bytes
- Making the fake controller truncate an answer to the reader's buffer, which a pseudo terminal cannot do, D3
- Checking the device's report sizes at run time from its report descriptor
- The traffic of the controller in `--trace`, declined as item 4 of "Beyond the request" of unit 0005

## Beyond the request

1. Taken into the unit as R10: an entry under "Amendments" in the `record.md` of unit 0005 that names this unit as
   the correction of R6 and of the report size of its document
2. Reading the size of the input report from the report descriptor in sysfs at each reading, rather than taking 384
   as a constant. It costs a parser of the descriptor's items and its tests. Without it a firmware that changed the
   size would still work as long as its reports are at least 96 bytes, and only reports longer than 384 bytes would be
   cut, of which nothing past byte 96 is used
