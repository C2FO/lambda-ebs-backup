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
