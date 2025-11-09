#!/bin/sh

export INJHOME=$(pwd)/injhome

rm -rf $INJHOME/ || echo "directory $INJHOME doesn't exist"

killall injectived
./../../../setup.sh
./../../../injectived.sh