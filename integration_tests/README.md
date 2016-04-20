# Integration Tests

## Running Tests

- To run integration tests:

`make integration` or `./test.sh test`

- To run a single integration test:

`./test.sh test test_name`

- To clean fixtures. Next time tests are run they will be rebuilt.

`make clean` or `./test.sh clean`

## Making Tests

### 1. Fixtures

Fixtures Directory: `integration_tests/fixtures`

- Folders in the fixtures directory will be built as docker images in alphabetical order
- Each folder should contain a `Dockerfile` and any resources necessary to build the image
- The resulting image will be named `cpfix_fixture_name`

*Note*: Since fixtures are created in alpha order, they can have FROM directives for previously created images

### 2. Tests

Tests Directory: `integration_tests/tests`

- Folders in the tests directory will be run as tests
- Each folder must contain `run.sh`
- The test folder can also contain `docker-compose.yml` for setting up the test environment and other resources it might need
- If `run.sh` returns success: `0` then the test passed, otherwise it failed

This script can make some assumptions:

- It's current directory PWD is the same as the script.
- Following environment variables are set:
  - `COMPOSE_FILE`
  - `COMPOSE_PROJECT_NAME` - The name of the test folder
  - `DOCKER_IP` - The IP of docker on the host machine
  - `CONTAINERPILOT_BIN` - Absolute path to containerpilot binary on the host.
- all fixtures in `integration_tests/fixtures` are created and are available as images

## How tests are executed

### Setup Test Fixtures
- Scan through all folders in `integration_tests/fixtures` in alpha order
- For each fixture:
  - Copy `build` into `integration_test/fixtures/fixture_name/build` so it can easily be sourced by the `Dockerfile`
  - `cd integration_tests/fixtures/fixture_name`
  - `docker build -t cpfix_fixture_name .`

### Run tests
- Scan through all folders in `integration_tests/tests` in alpha order
- For each test:
  - `cd integration_test/tests/test_name`
  - Run `docker-compose build`
  - Run `run.sh`
  - Stop/kill the compose environment

If *any* test fails, the test script will return a non-zero exit code.
