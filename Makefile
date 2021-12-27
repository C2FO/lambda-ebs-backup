test:
	go test -v -race ./...

create-iam-role:
	$(MAKE) -C cloudformation create-iam-role

update-iam-role:
	$(MAKE) -C cloudformation update-iam-role

clean:
	rm -f handler handler.zip lambda-ebs-backup

create-lambda-ebs-backup:
	$(MAKE) -C cloudformation create-lambda-ebs-backup

update-lambda-ebs-backup:
	$(MAKE) -C cloudformation update-lambda-ebs-backup

lambda-role-arn:
	@aws cloudformation describe-stacks --stack-name lambda-ebs-backup-role --output=json | jq -r '.Stacks | .[0] | .Outputs | .[0] | .OutputValue'

lambda-zip: clean
	CGO=0 GOOS=linux go build -o lambda-ebs-backup ./cmd/lambda-ebs-backup/main.go
	zip handler.zip ./lambda-ebs-backup
	aws s3 cp handler.zip s3://${S3_LAMBDA_BUCKET}/${S3_LAMBDA_KEY}

version-id:
	@aws s3api list-object-versions --output=json --bucket ${S3_LAMBDA_BUCKET} --prefix ${S3_LAMBDA_KEY} | jq -r '.Versions | .[0] | .VersionId'
