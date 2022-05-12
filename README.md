- [ipld-ethcl-indexer](#ipld-ethcl-indexer)
- [Running the Application](#running-the-application)
- [Development Patterns](#development-patterns)
  - [Logging](#logging)
  - [Testing](#testing)
- [Contribution](#contribution)
  - [Branching Structure](#branching-structure)

<small><i><a href='http://ecotrust-canada.github.io/markdown-toc/'>Table of contents generated with markdown-toc</a></i></small>

# ipld-ethcl-indexer

This application will capture all the `BeaconState`'s and `SignedBeaconBlock`'s from the consensus chain on Ethereum. This application is going to connect to the lighthouse client, but hypothetically speaking, it should be interchangeable with any eth2 beacon node.

To learn more about the applications individual components, please read the [application components](/application_component.md).

# Quick Start

## Running the Application

To run the application, do as follows:

1. Setup the prerequisite applications.
   a. Run a beacon client (such as lighthouse).
   b. Run a postgres DB.
   c. You can utilize the `stack-orchestrator` [repository](https://github.com/vulcanize/stack-orchestrato).

   ```
   ./wrapper.sh -e skip \
   -d ../docker/local/docker-compose-db.yml \
   -d ../docker/latest/docker-compose-lighthouse.yml \
   -v remove \
   -p ../local-config.sh

   ```

2. Run the start up command.

```
go run main.go capture head --db.address localhost \
  --db.password password \
  --db.port 8077 \
  --db.username vdbm \
  --db.name vulcanize_testing \
  --db.driver PGX \
  --bc.address localhost \
  --bc.port 5052 \
  --bc.connectionProtocol http \
  --log.level info \
  --log.output=true
```

## Running Tests

To run tests, you will need to clone another repository which contains all the ssz files.

1. `git clone git@github.com:vulcanize/ssz-data.git pkg/beaconclient/ssz-data`
2. To run unit tests, make sure you have a DB running: `make unit-test-local`
3. To run integration tests, make sure you have a lighthouse client and a DB running: `make integration-test-local-no-race` .

# Development Patterns

This section will cover some generic development patterns utilizes.

## Logging

For logging, please keep the following in mind:

- Utilize logrus.
- Use `log.Debug` to highlight that you are **about** to do something.
- Use `log.Info-Fatal` when the thing you were about to do has been completed, along with the result.

```
log.Debug("Adding 1 + 2")
a := 1 + 2
log.Info("1 + 2 successfully Added, outcome is: ", a)
```

- `loghelper.LogError(err)` is a pretty wrapper to output errors.

## Testing

This project utilizes `ginkgo` for testing. A few notes on testing:

- All tests within this code base will test **public methods only**.
- All test packages are named `{base_package}_test`. This ensures we only test the public methods.
- If there is a need to test a private method, please include why in the testing file.
- Unit tests must contain the `Label("unit")`.
- Unit tests should not rely on any running service (except for a postgres DB). If a running service is needed. Utilize an integration test.
- Integration tests must contain the `Label("integration")`.

#### Understanding Testing Components

A few notes about the testing components.

- The `TestEvents` map contains several events for testers to leverage when testing.
- Any object ending in `-dummy` is not a real object. You will also notice it has a present field called `MimicConfig`. This object will use an existing SSZ object, and update the parameters from the `Head` and `MimicConfig`.
  - This is done because creating an empty or minimal `SignedBeaconBlock` and `BeaconState` is fairly challenging.
  - By slightly modifying an existing object, we can test re-org, malformed objects, and other negative conditions.

# Contribution

If you want to contribute please make sure you do the following:

- Create a Github issue before starting your work.
- Follow the branching structure.
- Delete your branch once it has been merged.
  - Do not delete the `develop` branch. We can add branch protection once we make the branch public.

## Branching Structure

The branching structure is as follows: `main` <-- `develop` <-- `your-branch`.

It is adviced that `your-branch` follows the following structure: `{type}/{issue-number}-{description}`.

- `type` - This can be anything identifying the reason for this PR, for example: `bug`, `feature`, `release`.
- `issue-number` - This is the issue number of the GitHub issue. It will help users easily find a full description of the issue you are trying to solve.
- `description` - A few words to identify your issue.
