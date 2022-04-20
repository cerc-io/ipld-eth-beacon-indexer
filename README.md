# ipld-ethcl-indexer

This application will capture all the `BeaconState`'s and `SignedBeaconBlock`'s from the consensus chain on Ethereum.

# Running the Application

To run the application, utilize the following command, and update the values as needed.

```
go run main.go capture head --db.address localhost \
  --db.password password \
  --db.port 8077 \
  --db.username username \
  --lh.address localhost \
  --lh.port 5052
```

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
