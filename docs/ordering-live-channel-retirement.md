# Ordering a live channel retirement

The live channel is never deleted before the shared path is proven, because the proof is what makes the deletion safe.

This repository retired its own notification channel on 2026-09-14. That retirement did **not**
satisfy the rule above. The operator waived it explicitly, on the same date, rather than meet it.
The rule still stands for the next retirement: removing a channel that is still carrying traffic
before the replacement has been seen delivering is how a push goes silent without an error, a
failed deploy, or any other signal.

What this retirement rests on instead is code evidence:

- The shared publish path shipped and was released in this repository.
- The shared core's end-to-end delivery was proven on real production traffic by the producer that
  landed it.

What that evidence does not cover, and what was therefore not verified here:

- This watcher has never been observed publishing through the shared core from a deployed host.
- The deployed config's OpenClaw wake entry is removed in the same change, so that wake path stops
  here rather than migrating to the shared core.

The rollback is the previous binary and the previous config on the host, together with this
repository's history.
