# lambda-ebs-backup

Lambda EBS Backup uses labmda function to manage EBS snapshots on a user-defined schedule.

<!-- TOC depthFrom:1 depthTo:6 withLinks:1 updateOnSave:1 orderedList:0 -->

- [lambda-ebs-backup](#lambda-ebs-backup)
	- [Usage](#usage)
		- [Creating labmda-ebs-backup IAM Role](#creating-labmda-ebs-backup-iam-role)
		- [Uploading the code to S3](#uploading-the-code-to-s3)
		- [Creating the Lambda Function](#creating-the-lambda-function)

<!-- /TOC -->

## Usage

### Creating labmda-ebs-backup IAM Role

To create the lambda function which cmanages backups, first you must
create the role which allows it to create backups. You can do this with:

```sh
make create-iam-role
```

If you have already created the IAM role and need to update it, you
can do so by running:

```sh
make update-iam-role
```

Finally, capture the lambda role arn like so:

```sh
export LAMBDA_ROLE_ARN=$(make lambda-role-arn)
```

### Uploading the code to S3

After you create the IAM role, you'll need to upload the code to S3 so it is
available to the cloudformation to use for the lambda function. You can do so
with:

```sh
export S3_LAMBDA_BUCKET="your-bucket"
export S3_LAMBDA_KEY="your-key.zip"
make lambda-zip
export S3_OBJECT_VERSION_ID=$(make version-id)
```

Now `S3_OBJECT_VERSION_ID` contains the version ID of the S3 object in S3
which contains your code.

### Creating the Lambda Function

Finally, you can create the lambda function like so:

```
make create-lambda-ebs-backup
```

### Tagging Volumes for backup

In order to backup a volume, you must opt-in by setting a tag key:value pair on
the volume. By default, this is `lambda-ebs-backup/backup:true`. This tells the
lambda function that we are to create a snapshot of the volume.


#### Controlling how many Volume Snapshots are kept

By default, only 2 volume snapshots are kept. You can override this globally
by setting the environment variable `DEFAULT_MAX_KEEP_SNAPSHOTS` in the
cloudformation for the lambda function. Alternatively, you can control this on
a volume by volume basis by setting the following tag key:value pair on the
volume: `lambda-ebs-backup/max-keep-snapshots`.

### Tagging Images for AMI creation

In order to take an AMI of an instance, you must opt-in by setting a tag
key:value pair on the instance. By default, this is
`lambda-ebs-backup/image:true`. This tells the lambda function that we should
create an AMI of this instance.

#### Controlling how many AMIs are kept

By default, only 2 AMIs are kept. You can override this globally
by setting the environment variable `DEFAULT_MAX_KEEP_IMAGES` in the
cloudformation for the lambda function. Alternatively, you can control this on
an instance by instance basis by setting the following tag key:value pair on the
instance: `lambda-ebs-backup/max-keep-images`.

#### Controlling reboot behvior for images.

By default, we will reboot the instance when creating an image of it. However,
you might prefer that the instance does not reboot and wish to chance it on
filesystem integrity. You can allow this by setting the tag key:value pair to
`lambda-ebs-backup/reboot-on-image:false` on your instance.
