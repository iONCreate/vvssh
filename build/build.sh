#!/usr/bin/env bash
SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ]; do # resolve $SOURCE until the file is no longer a symlink
  DIR="$( cd -P "$( dirname "$SOURCE" )" && pwd )"
  SOURCE="$(readlink "$SOURCE")"
  [[ $SOURCE != /* ]] && SOURCE="$DIR/$SOURCE" # if $SOURCE was a relative symlink, we need to resolve it relative to the path where the symlink file was located
done
RUNDIR="$( cd -P "$( dirname "$SOURCE" )" && pwd )"

GOPATH="$( cd -P "$( dirname "$SOURCE/../" )" && pwd )"

echo ${GOPATH}

go build -i -o ${GOPATH}/out/vvssh ${GOPATH}/src/ioncreate.com/tools/vvssh/main/vvssh.go