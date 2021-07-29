#!/bin/sh

if [ $# -ne 2 ]
then
	echo [username] and [password] required
	exit 1
fi

username=$1
password=$2

access_token=`curl \
	--header "Content-Type: application/x-www-form-urlencoded" \
	--request POST \
	--data "grant_type=password&username=${username}&password=${password}" \
   	--user "ui:uiman" \
  	http://localhost:8080/uat/sso/oauth/token || exit $? | jq -r .access_token` exit 1 2> /dev/null

curl --header "Authorization: Bearer $access_token" --request POST http://localhost:8080/uat/sso/me/apitoken | jq -r .access_token
