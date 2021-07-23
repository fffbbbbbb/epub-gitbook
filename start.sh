#!/bin/sh

/build/main

cd /build/output

gitbook init

echo -e '{\n"plugins" : [ "include-html" ]\n}' > book.json

npm i gitbook-plugin-include-html

gitbook serve