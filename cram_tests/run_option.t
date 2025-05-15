Setup

  $ . "$TESTDIR"/setup.sh

Run buildenv with -r

  $ be -r "echo -n hi"
  hi (no-eol)

Bad command

  $ be -r "./notacommand" 2>/dev/null
  [127]

Vars are there

  $ be -r "echo \${TEST}"
  no secrets
