Setup

  $ . "$TESTDIR"/setup.sh

Make env vars

  $ export INP=$(printf '{"VAR1": "VAL1", "VAR2": "VAL2"}' | base64)
  $ be -v -f /dev/null -u INP -x > codec_test.blob
  $ export BLOB=`cat codec_test.blob`
  $ echo "$BLOB"
  eyJWQVIxIjoiVkFMMSIsIlZBUjIiOiJWQUwyIn0=
  $ be -v -f /dev/null -u BLOB
  export VAR1="VAL1"
  export VAR2="VAL2"

