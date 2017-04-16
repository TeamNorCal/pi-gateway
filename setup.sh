#!/bin/sh
export GOPATH=`pwd`
export GOBIN=`pwd`/bin
export PATH=`pwd`/bin:$PATH
export LOGXI=\*
export LOGXI_FORMAT=happy,maxcol=1024
export LOGXI_COLORS=key=cyan+h,value,misc=blue+h,source=magenta,TRC,DBG,WRN=yellow,INF=green,ERR=red+h
