# lambda-ebs-backup

Lambda EBS Backup uses labmda function to manage EBS snapshots on a user-defined schedule.

## Usage

### Creating labmda-ebs-backup IAM Role

To create the lambda function which cmanages backups, first you must
create the role which allows it to create backups. You can do this with:

```
$ make create-iam-role
```

If you have already created the IAM role and need to update it, you
can do so by running:

```
$ make update-iam-role
```
