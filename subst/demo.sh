#!/bin/bash

set -v

echo '{"deliver":"{?want}"}' | plaxbs -p '?want="tacos"'

echo 'I like {?want|text}.' | plaxbs -p '?want="tacos"'

echo '{"deliver":"{?want}"}' | plaxbs -p '?want=["tacos","chips"]'

echo '{"deliver":["beer","{?want|json$}"]}' | plaxbs -p '?want=["tacos","chips"]'

echo '{"deliver":"{?want}","n":{?want | js $.length | json}}' | plaxbs -p '?want=["tacos","chips"]'

echo '{"deliver":"{?want | jq .[0] | json}"}' | plaxbs -p '?want=["tacos","chips"]'

echo 'The order: {?want|text$}.' | plaxbs -p '?want=["tacos","chips"]'

echo 'The first item: {?want|jq .[0]|text}.' | plaxbs -p '?want=["tacos","chips"]'

echo '{"deliver":{"chips":2,"":"{?want|json@}"}}' |
    plaxbs -p '?want={"tacos":2,"salsa":1}' -check-json-in -check-json-out

echo 'I want <?want|text>.' | plaxbs -d "<>" -p '?want="tacos"'

echo '{"deliver":"?want"}' | plaxbs -bind -p '?want={"tacos":3}'
