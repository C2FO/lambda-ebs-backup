CF = aws cloudformation
CREATE = create-stack
UPDATE = update-stack
PREVENT_DELETE = --enable-termination-protection
ALLOW_IAM = --capabilities CAPABILITY_NAMED_IAM

create-iam-role:
	$(CF) $(CREATE) --stack-name lambda-ebs-backup-role \
			--template-body file://$$(pwd)/iam.yaml \
			$(PREVENT_DELETE) $(ALLOW_IAM)

update-iam-role:
	$(CF) $(UPDATE) --stack-name lambda-ebs-backup-role \
			--template-body file://$$(pwd)/iam.yaml \
			$(ALLOW_IAM)

create-lambda-ebs-backup:
	$(CF) $(CREATE) --stack-name lambda-ebs-backup \
			--template-body file://$$(pwd)/lambda.yaml \
			$(PREVENT_DELETE) \
			--parameters ParameterKey=S3Bucket,ParameterValue=${S3_LAMBDA_BUCKET} \
				     ParameterKey=S3Key,ParameterValue=${S3_LAMBDA_KEY} \
				     ParameterKey=S3ObjectVersionId,ParameterValue=${S3_OBJECT_VERSION_ID} \
				     ParameterKey=LambdaRoleArn,ParameterValue=${LAMBDA_ROLE_ARN}


update-lambda-ebs-backup:
	$(CF) $(UPDATE) --stack-name lambda-ebs-backup \
			--template-body file://$$(pwd)/lambda.yaml \
			--parameters ParameterKey=S3Bucket,ParameterValue=${S3_LAMBDA_BUCKET} \
 				     ParameterKey=S3Key,ParameterValue=${S3_LAMBDA_KEY} \
				     ParameterKey=S3ObjectVersionId,ParameterValue=${S3_OBJECT_VERSION_ID} \
				     ParameterKey=LambdaRoleArn,ParameterValue=${LAMBDA_ROLE_ARN}
