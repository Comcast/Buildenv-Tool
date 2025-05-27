Setup

  $ . "$TESTDIR"/setup.sh

Try injecting quote

  $ echo '{"vars":{"Q": "\"; echo bad \""}}' > test.yml
  $ be -f test.yml
  export Q="\"; echo bad \""

Bad keys

  $ echo '{"vars":{"export hi=there; dosomethingevil && ": "hi"}}' > test2.yml
  $ be -f test2.yml
