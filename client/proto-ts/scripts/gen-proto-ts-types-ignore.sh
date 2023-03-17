#!/bin/bash

search1="getExtension():"
replace1="// @ts-ignore \n  getExtension():"
search2="setExtension("
replace2="// @ts-ignore \n  setExtension("

FILES=$( find ./client/proto-ts/gen -type f -name '*.d.ts' )

for file in $FILES
do  
	sed -ie "s/${search1//\//\\/}/${replace1//\//\\/}/g" $file
  sed -ie "s/${search2//\//\\/}/${replace2//\//\\/}/g" $file
done

FILES=$( find ./client/proto-ts/gen -type f -name '*.d.tse' )

for file in $FILES
do
	rm -f $file
done
