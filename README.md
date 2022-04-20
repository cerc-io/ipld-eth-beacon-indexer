# ipld-ethcl-indexer

This application will capture all the `BeaconState`'s and `SignedBeaconBlock`'s from the consensus chain on Ethereum. This application is going to connect to the lighthouse client, but hypothetically speaking, it should be interchangeable with any eth2 beacon node.

# Running the Application

To run the application, utilize the following command, and update the values as needed.

```
go run main.go capture head --db.address localhost \
  --db.password password \
  --db.port 8077 \
  --db.username vdbm \
  --db.name vulcanize_testing \
  --db.driver PGX \
  --bc.address localhost \
  --bc.port 5052 \
  --log.level info
```

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

## Boot

The boot package in `internal` is utilized to start the application. Everything in the boot process must complete successfully for the application to start. If it does not, the application will not start.

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
