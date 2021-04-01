#!/bin/bash

set -v

echo '{"deliver":"{?want}"}' | plaxsubst -p '?want="tacos"'

echo 'I like {?want|text}.' | plaxsubst -p '?want="tacos"'

echo '{"deliver":"{?want}"}' | plaxsubst -p '?want=["tacos","chips"]'

echo '{"deliver":["beer","{?want|json$}"]}' | plaxsubst -p '?want=["tacos","chips"]'

echo '{"deliver":"{?want}","n":{?want | js $.length | json}}' | plaxsubst -p '?want=["tacos","chips"]'

echo '{"deliver":"{?want | jq .[0] | json}"}' | plaxsubst -p '?want=["tacos","chips"]'

echo 'The order: {?want|text$}.' | plaxsubst -p '?want=["tacos","chips"]'

echo 'The first item: {?want|jq .[0]|text}.' | plaxsubst -p '?want=["tacos","chips"]'

echo '{"deliver":{"chips":2,"":"{?want|json@}"}}' |
    plaxsubst -p '?want={"tacos":2,"salsa":1}' -check-json-in -check-json-out

echo 'I want <?want|text>.' | plaxsubst -d "<>" -p '?want="tacos"'

echo '{"deliver":"?want"}' | plaxsubst -bind -p '?want={"tacos":3}'

echo '{"deliver":"?want | jq .[0]"}' | plaxsubst -bind -p '?want=[{"tacos":3},{"queso":1}]'

