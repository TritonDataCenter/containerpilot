#!/bin/bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )/.." && pwd )"

DEBUG_LOG=${DEBUG_LOG:-"/dev/null"}
ROOT_DIR="$DIR/integration_tests"
FIXTURE_DIR="$ROOT_DIR/fixtures"
TESTS_DIR="$ROOT_DIR/tests"

if [ "$DEBUG_LOG" != "/dev/null" ]; then
  DEBUG_LOG_DIR="$( cd "$( dirname "$DEBUG_LOG" )" && pwd )"
  DEBUG_LOG="$DEBUG_LOG_DIR/$(basename $DEBUG_LOG)"
fi

die() {
  echo "$@" | tee -a ${DEBUG_LOG} >&2
  exit 1
}

log() {
  echo "$(date '+%Y/%m/%d %H:%M:%S')" "$@" | tee -a ${DEBUG_LOG}
}

debug() {
  echo "DEBUG - $(date '+%Y/%m/%d %H:%M:%S')" "$@" >> ${DEBUG_LOG}
}

banner() {
  log "==== ${1} ===="
}

docker_running() {
  local id=$(docker ps -q --filter="name=${1}")
  [ -n "$id" ]
}

## Sanity Check
if [ ! -d $FIXTURE_DIR ]; then die "Unable to find fixtures: $FIXTURE_DIR"; fi
if [ ! -d $TESTS_DIR ];   then die "Unable to find tests: $TESTS_DIR"; fi

export FIXTURE_PREFIX=${FIXTURE_PREFIX:-"cpfix_"}
debug "FIXTURE_PREFIX=$FIXTURE_PREFIX"

debug "ROOT_DIR=$ROOT_DIR"
debug "FIXTURE_DIR=$FIXTURE_DIR"
debug "TESTS_DIR=$TESTS_DIR"

## Main Code

create_test_fixtures() {
  banner "Create Test Fixtures"
  if [ ! -f "$DIR/build/containerpilot" ]; then die "ContainerPilot not built. Did you make?"; fi
  find $FIXTURE_DIR -maxdepth 1 -mindepth 1 -type d | \
    sort | \
    while read FDIR; do
      FNAME="${FIXTURE_PREFIX}$(basename $FDIR)"
      rm -rf $FDIR/build
      cp -r $DIR/build $FDIR/build
      cd $FDIR
      if [ -z "$(docker images -q $FNAME)" ]; then
        log "Create fixture $FNAME"
        docker build -t $FNAME --no-cache=false . 2>&1 >> ${DEBUG_LOG}
        rm -rf $FDIR/build
      else
        log "Skipping Fixture $FNAME ... already exists"
      fi
  done
}

destroy_test_fixtures() {
    banner "Destroying Test Fixtures"
    find $FIXTURE_DIR -maxdepth 1 -mindepth 1 -type d |\
      while read FDIR; do
        FNAME="${FIXTURE_PREFIX}$(basename $FDIR)"
        if [ -n "$(docker images -q $FNAME)" ]; then
          log "Remove Fixture fixture $FNAME"
          docker ps -q --filter "ancestor=${FNAME}" | while read CONTAINER_ID; do
            log " - Killing $CONTAINER_ID"
            docker rm --force $CONTAINER_ID >> ${DEBUG_LOG} 2>&1
          done
          docker rmi --force $FNAME >> ${DEBUG_LOG} 2>&1
          rm -rf $FDIR/build
        fi
    done
}

build_test_compose() {
  if [ -r "$COMPOSE_FILE" ]; then
    docker-compose build >> ${DEBUG_LOG} 2>&1
  fi
}

destroy_test_compose() {
  if [ -r "$COMPOSE_FILE" ]; then
    docker-compose kill >> ${DEBUG_LOG} 2>&1
    docker-compose rm -f >> ${DEBUG_LOG} 2>&1
  fi
}

run_test() {
  TDIR=$1
  if [ ! -r "$TDIR/run.sh" ]; then
    log "WARN - No run.sh found in $TDIR"
  else
    cd $TDIR
    export COMPOSE_PROJECT_NAME="$(basename $TDIR)"
    export COMPOSE_FILE="./docker-compose.yml"
    export CONTAINERPILOT_BIN="$DIR/build/containerpilot"
    log "TEST: $COMPOSE_PROJECT_NAME"
    build_test_compose
    if ./run.sh; then
      log "PASS: $COMPOSE_PROJECT_NAME"
    else
      log "FAIL: $COMPOSE_PROJECT_NAME"
      echo $COMPOSE_PROJECT_NAME >> "$FAILED"
    fi
    destroy_test_compose
  fi
}

run_tests() {
  banner "Running Integration Tests"
  FAILED=$TESTS_DIR/failed.log
  rm -f $FAILED
  TARGET="$1"
  if [ -z "$TARGET" ] || [ "$TARGET" = "all" ]; then
    ## Run All Tests
    for TDIR in $(find $TESTS_DIR -maxdepth 1 -mindepth 1 -type d ); do
      run_test $TDIR
    done
  else
    ## Run specific test
    TDIR=$(find $TESTS_DIR -maxdepth 1 -mindepth 1 -type d -name $TARGET)
    run_test $TDIR
  fi
  [ ! -r "$FAILED" ]
}

clean_tests() {
  find $TESTS_DIR -maxdepth 1 -mindepth 1 -type d |\
    while read TDIR; do
      cd $TDIR
      export COMPOSE_PROJECT_NAME="$(basename $TDIR)"
      export COMPOSE_FILE="./docker-compose.yml"
      destroy_test_compose
    done
}

COMMAND=${1:-"test"}
shift
case $COMMAND in
  create_test_fixtures)
    create_test_fixtures
  ;;
  test)
    create_test_fixtures
    run_tests "$1"
    result=$?
    if [ $result -ne 0 ]; then
      banner "Tests Failed!"
      exit 1
    fi
  ;;
  clean)
    destroy_test_fixtures
    clean_tests
  ;;
esac
