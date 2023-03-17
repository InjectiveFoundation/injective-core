#!/bin/bash

search1="@improbable-eng/grpc-web"
replace1="@injectivelabs/grpc-web"

FILES=$( find ./client/proto-ts/gen -type f )

for file in $FILES
do  
	sed -ie "s/${search1//\//\\/}/${replace1//\//\\/}/g" $file
done
