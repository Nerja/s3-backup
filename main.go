package main

import (
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/backup"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/iam"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/s3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		assumeRole, err := iam.GetPolicyDocument(ctx, &iam.GetPolicyDocumentArgs{
			Statements: []iam.GetPolicyDocumentStatement{
				{
					Effect: pulumi.StringRef("Allow"),
					Principals: []iam.GetPolicyDocumentStatementPrincipal{
						{
							Type: "Service",
							Identifiers: []string{
								"backup.amazonaws.com",
							},
						},
					},
					Actions: []string{
						"sts:AssumeRole",
					},
				},
			},
		}, nil)
		if err != nil {
			return err
		}

		backupRole, err := iam.NewRole(ctx, "exampleRole", &iam.RoleArgs{
			AssumeRolePolicy: pulumi.String(assumeRole.Json),
		})
		if err != nil {
			return err
		}
		_, err = iam.NewRolePolicyAttachment(ctx, "exampleRolePolicyAttachment", &iam.RolePolicyAttachmentArgs{
			PolicyArn: pulumi.String("arn:aws:iam::aws:policy/AWSBackupServiceRolePolicyForS3Backup"),
			Role:      backupRole.Name,
		})
		if err != nil {
			return err
		}

		vault, err := backup.NewVault(ctx, "backup-vault", &backup.VaultArgs{})
		if err != nil {
			return err
		}

		plan, err := backup.NewPlan(ctx, "backup", &backup.PlanArgs{
			Rules: backup.PlanRuleArray{
				&backup.PlanRuleArgs{
					RuleName:        pulumi.String("hourly-backup"),
					Schedule:        pulumi.String("cron(0 * * * ? *)"),
					TargetVaultName: vault.ID(),
					Lifecycle: &backup.PlanRuleLifecycleArgs{
						DeleteAfter: pulumi.Int(14),
					},
				},
			},
		})
		if err != nil {
			return err
		}

		_, err = backup.NewSelection(ctx, "backup-selection", &backup.SelectionArgs{
			PlanId: plan.ID(),
			SelectionTags: backup.SelectionSelectionTagArray{
				&backup.SelectionSelectionTagArgs{
					Type:  pulumi.String("STRINGEQUALS"),
					Key:   pulumi.String("aws-backup-name"),
					Value: plan.ID(),
				},
			},
			IamRoleArn: backupRole.Arn,
		})
		if err != nil {
			return err
		}

		bucket, err := s3.NewBucket(ctx, "bucket-to-backup", &s3.BucketArgs{
			Versioning: &s3.BucketVersioningArgs{
				Enabled: pulumi.Bool(true),
			},
			LifecycleRules: s3.BucketLifecycleRuleArray{
				&s3.BucketLifecycleRuleArgs{
					Enabled: pulumi.Bool(true),
					NoncurrentVersionExpiration: &s3.BucketLifecycleRuleNoncurrentVersionExpirationArgs{
						Days: pulumi.Int(30),
					},
					Expiration: &s3.BucketLifecycleRuleExpirationArgs{
						Days:                      pulumi.Int(30),
						ExpiredObjectDeleteMarker: pulumi.Bool(true),
					},
					AbortIncompleteMultipartUploadDays: pulumi.Int(30),
				},
			},
			Tags: pulumi.StringMap{
				"aws-backup-name": plan.ID(),
			},
		})
		if err != nil {
			return err
		}

		ctx.Export("bucketName", bucket.ID())
		return nil
	})
}
