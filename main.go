package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	LockTimeoutVar    = "timeout"
	LockTableVar      = "table"
	LockKeyNameVar    = "key"
	LockNameVar       = "name"
	LockIdentifierVar = "identifier"

	DefaultLockTimeout    = 30
	DefaultLockTable      = "github-action-locks"
	DefaultLockKeyName    = "LockID"
	DefaultLockName       = "testing"
	DefaultLockIdentifier = ""
)

func createLock(ctx context.Context, svc *dynamodb.DynamoDB, lockTable, lockKeyName, lockName, lockIdentifier string) error {
	_, err := svc.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(lockTable),
		Item: map[string]*dynamodb.AttributeValue{
			lockKeyName: {
				S: aws.String(lockName),
			},
			LockIdentifierVar: {
				S: aws.String(lockIdentifier),
			},
		},
		ConditionExpression: aws.String(fmt.Sprintf("attribute_not_exists(%s)", lockKeyName)),
	})
	return err
}

func lock() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lock",
		Short: "Create a lock",
		Run: func(_ *cobra.Command, _ []string) {
			LockTimeout := viper.GetInt(LockTimeoutVar)
			LockTable := viper.GetString(LockTableVar)
			LockKeyName := viper.GetString(LockKeyNameVar)
			LockName := viper.GetString(LockNameVar)
			LockIdentifier := viper.GetString(LockIdentifierVar)

			log.Printf("LockTimeout: %v", LockTimeout)
			log.Printf("LockTable: %v", LockTable)
			log.Printf("LockKeyName: %v", LockKeyName)
			log.Printf("LockName: %v", LockName)
			log.Printf("LockIdentifier: %v", LockIdentifier)

			svc := dynamodb.New(session.Must(session.NewSession()))
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(LockTimeout)*time.Minute)
			defer cancel()

			// AcquireLock
			log.Println("Acquiring lock")

			for {
				err := createLock(ctx, svc, LockTable, LockKeyName, LockName, LockIdentifier)

				if err != nil {
					if aerr, ok := err.(awserr.Error); ok {
						if aerr.Code() != dynamodb.ErrCodeConditionalCheckFailedException {
							log.Fatalf("Failed to create lock: %+v", err)
						}
					}
				}

				select {
				case <-ctx.Done():
					log.Fatal("Timed out waiting to acquire lock")
				case <-time.After(5 * time.Second):
					log.Print("Checking again")
				}
			}
		},
	}

	cmd.PersistentFlags().Int(LockTimeoutVar, DefaultLockTimeout, "How long to wait and retry acquiring a lock if it's already been acquired. Value in minutes")
	viper.BindPFlag(LockTimeoutVar, cmd.PersistentFlags().Lookup(LockTimeoutVar))

	cmd.PersistentFlags().String(LockTableVar, DefaultLockTable, "The name of the DynamoDB table to create the lock in")
	viper.BindPFlag(LockTableVar, cmd.PersistentFlags().Lookup(LockTableVar))

	cmd.PersistentFlags().String(LockKeyNameVar, DefaultLockKeyName, "The name of the column in the DynamoDB table to use for storing the lock name")
	viper.BindPFlag(LockKeyNameVar, cmd.PersistentFlags().Lookup(LockKeyNameVar))

	cmd.PersistentFlags().String(LockNameVar, DefaultLockName, "The name of the DynamoDB table to create the lock in")
	viper.BindPFlag(LockNameVar, cmd.PersistentFlags().Lookup(LockNameVar))

	cmd.PersistentFlags().String(LockIdentifierVar, DefaultLockIdentifier, "The name of the DynamoDB table to create the lock in")
	viper.BindPFlag(LockIdentifierVar, cmd.PersistentFlags().Lookup(LockIdentifierVar))

	return cmd
}

func unlock() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unlock",
		Short: "Release a lock",
		Run: func(_ *cobra.Command, _ []string) {
			LockTimeout := viper.GetInt(LockTimeoutVar)
			LockTable := viper.GetString(LockTableVar)
			LockKeyName := viper.GetString(LockKeyNameVar)
			LockName := viper.GetString(LockNameVar)
			LockIdentifier := viper.GetString(LockIdentifierVar)

			log.Printf("LockTimeout: %v", LockTimeout)
			log.Printf("LockTable: %v", LockTable)
			log.Printf("LockKeyName: %v", LockKeyName)
			log.Printf("LockName: %v", LockName)
			log.Printf("LockIdentifier: %v", LockIdentifier)

			svc := dynamodb.New(session.Must(session.NewSession()))
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(LockTimeout)*time.Minute)
			defer cancel()

			log.Println("Acquiring lock to release it")
			output, err := svc.GetItemWithContext(ctx, &dynamodb.GetItemInput{
				TableName:      aws.String(LockTable),
				ConsistentRead: aws.Bool(true),
				Key: map[string]*dynamodb.AttributeValue{
					LockKeyName: {
						S: aws.String(LockName),
					},
				},
			})
			if err != nil {
				log.Fatal(err)
			}

			if len(output.Item) == 0 {
				log.Print("No lock exists to unlock")
			} else {
				log.Print("Releasing lock")

				_, err := svc.DeleteItem(&dynamodb.DeleteItemInput{
					TableName: aws.String(LockTable),
					Key: map[string]*dynamodb.AttributeValue{
						LockKeyName: {
							S: aws.String(LockName),
						},
					},
				})
				if err != nil {
					log.Fatal(err)
				}
			}
		},
	}

	cmd.PersistentFlags().Int(LockTimeoutVar, DefaultLockTimeout, "How long to wait and retry acquiring a lock if it's already been acquired. Value in minutes")
	viper.BindPFlag(LockTimeoutVar, cmd.PersistentFlags().Lookup(LockTimeoutVar))

	cmd.PersistentFlags().String(LockTableVar, DefaultLockTable, "The name of the DynamoDB table to create the lock in")
	viper.BindPFlag(LockTableVar, cmd.PersistentFlags().Lookup(LockTableVar))

	cmd.PersistentFlags().String(LockKeyNameVar, DefaultLockKeyName, "The name of the column in the DynamoDB table to use for storing the lock name")
	viper.BindPFlag(LockKeyNameVar, cmd.PersistentFlags().Lookup(LockKeyNameVar))

	cmd.PersistentFlags().String(LockNameVar, DefaultLockName, "The name of the DynamoDB table to create the lock in")
	viper.BindPFlag(LockNameVar, cmd.PersistentFlags().Lookup(LockNameVar))

	cmd.PersistentFlags().String(LockIdentifierVar, DefaultLockIdentifier, "The name of the DynamoDB table to create the lock in")
	viper.BindPFlag(LockIdentifierVar, cmd.PersistentFlags().Lookup(LockIdentifierVar))

	return cmd
}

func main() {

	var (
		rootCmd = &cobra.Command{
			Use:   "github-action-locks",
			Short: "Create a distributed lock for a GitHub Action",
		}
	)

	viper.SetEnvPrefix("INPUT")
	viper.AutomaticEnv()

	rootCmd.AddCommand(lock())
	rootCmd.AddCommand(unlock())

	rootCmd.Execute()
}
