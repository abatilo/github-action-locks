# github-action-locks

[![Main](https://github.com/abatilo/github-action-locks/workflows/Main/badge.svg)](https://github.com/abatilo/github-action-locks/actions?query=workflow%3AMain)
[![license](https://img.shields.io/github/license/abatilo/github-action-locks.svg)](https://github.com/abatilo/github-action-locks/blob/master/LICENSE)

Guarantee atomic execution of your GitHub Action workflows. Why would you want
to do that, you might ask?

The reason I built this GitHub Action is specifically because [Pulumi doesn't
support locking remote state](https://github.com/pulumi/pulumi/pull/2697)
unless you use their SaaS offering. I deploy various bits of infrastructure
using the [Official Pulumi GitHub
Actions](https://www.pulumi.com/docs/guides/continuous-delivery/github-actions/)
and I want to be able to fearlessly merge code and watch as the updates are
pushed one at a time.

## Getting started

`github-action-locks` works by creating a record in a DynamoDB table. We rely
on [Conditional
Writes](https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/WorkingWithItems.html#WorkingWithItems.ConditionalUpdate)
in order to guarantee that we can't write or create the same lock record twice.

Additionally, this action works by utilizing the
[post-entrypoint](https://help.github.com/en/actions/creating-actions/metadata-syntax-for-github-actions#post-entrypoint)
functionality of GitHub Actions. That is to say that as long as you start this
action early in your workflow, the lock will get cleaned up at the end of the
job execution once all of the "post" Actions are invoked.

### DynamoDB table

Here is a minimal Terraform example of what a DynamoDB table should look like
for usage with this Action:
```
resource "aws_dynamodb_table" "github-action-locks-table" {
  name           = "github-action-locks"
  billing_mode   = "PAY_PER_REQUEST"
  hash_key       = "LockID"
  attribute {
    name = "LockID"
    type = "S"
  }
}
```

### IAM Permissions

Here is the minimum IAM Policy required for `github-action-locks` to work:
```
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "dynamodb:GetItem",
                "dynamodb:PutItem",
                "dynamodb:DeleteItem"
            ],
            "Resource": "arn:aws:dynamodb:*:*:table/github-action-locks"
        }
    ]
}
```

### Required Environment Variables
Since we're connecting to AWS, you must set the required AWS variables for
creating a session as needed by the Go AWS SDK which are `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, and `AWS_REGION`. These variables will be used to create the DynamoDB client which create the locks.

### Additional Configuration
There are 4 input variables that you can use to control the behavior of this action:

| Input     | Description                                    | Default               |
| -----     | -----------                                    | -------               |
| `timeout` | How long to wait to acquire a lock, in minutes | `30`                  |
| `table`   | DynamoDB table to write the lock in            | `github-action-locks` |
| `key`     | Name of the column where we write locks        | `LockID`              |
| `name`    | Name of the lock                               | `foobar`              |

See [action.yml](action.yml) for more information.

## Example workflow

This workflow uses the workflow name as the identifier for the lock. You can
use any value you want here to synchronize Actions within the same repo,
workflows with multiple parallel jobs, or even Actions across repositories.

```yaml
name: Main
on: [push, pull_request]

env:
  AWS_ACCESS_KEY_ID: ${{ secrets.AWS_ACCESS_KEY_ID }}
  AWS_SECRET_ACCESS_KEY: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
  AWS_REGION: ${{ secrets.AWS_REGION }}
jobs:
  test:
    strategy:
      fail-fast: false
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@master
    - name: Create lock
      uses: abatilo/github-action-locks@v1
      with:
        timeout: "30"
        table: "github-action-locks"
        key: "LockID"
        name: "${{ github.workflow }}" # Use the workflow name, in this case "Main" as the lock identifier
    - run: |
        echo "Do something that takes a long time here"
        sleep 5
```
