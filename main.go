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
	// LockTimeoutVar is the key for the setting to control how long to wait to acquire a lock, in minutes
	LockTimeoutVar = "timeout"

	// LockTableVar is the key for the setting to control the DynamoDB table to write the lock in
	LockTableVar = "table"

	// LockKeyNameVar is the key for the setting to control the name of the column where we write locks
	LockKeyNameVar = "key"

	// LockNameVar is the key for the setting to control the name of the lock
	LockNameVar = "name"

	// DefaultLockTimeout is the default time, in minutes, for how long to wait to acquire a lock before giving up
	DefaultLockTimeout = 30

	// DefaultLockTable is the default name of the DynamoDB table to write the lock in
	DefaultLockTable = "github-action-locks"

	// DefaultLockKeyName is the default name of the column where we write locks
	DefaultLockKeyName = "LockID"

	// DefaultLockName is the default name for the lock to create
	DefaultLockName = "foobar"
)

func lock() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lock",
		Short: "Create a lock",
		Run: func(_ *cobra.Command, _ []string) {
			LockTimeout := viper.GetInt(LockTimeoutVar)
			LockTable := viper.GetString(LockTableVar)
			LockKeyName := viper.GetString(LockKeyNameVar)
			LockName := viper.GetString(LockNameVar)

			log.Print("Creating lock with the following parameters:")
			log.Printf("LockTimeout: %v", LockTimeout)
			log.Printf("LockTable: %v", LockTable)
			log.Printf("LockKeyName: %v", LockKeyName)
			log.Printf("LockName: %v", LockName)

			svc := dynamodb.New(session.Must(session.NewSession()))
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(LockTimeout)*time.Minute)
			defer cancel()

			log.Println("Acquiring lock")
			for {
				_, err := svc.PutItem(&dynamodb.PutItemInput{
					TableName: aws.String(LockTable),
					Item: map[string]*dynamodb.AttributeValue{
						LockKeyName: {
							S: aws.String(LockName),
						},
					},
					ConditionExpression: aws.String(fmt.Sprintf("attribute_not_exists(%s)", LockKeyName)),
				})

				if err != nil {
					if aerr, ok := err.(awserr.Error); ok {
						if aerr.Code() != dynamodb.ErrCodeConditionalCheckFailedException {
							log.Fatalf("Failed to create lock: %+v", err)
						}
					}
				} else {
					log.Printf("Lock acquired")
					return
				}

				select {
				case <-ctx.Done():
					log.Fatal("Timed out waiting to acquire lock")
				case <-time.After(5 * time.Second):
					log.Print("Failed to acquire lock, trying again")
				}
			}
		},
	}

	cmd.PersistentFlags().Int(LockTimeoutVar, DefaultLockTimeout, "How long to wait to acquire a lock, in minutes")
	viper.BindPFlag(LockTimeoutVar, cmd.PersistentFlags().Lookup(LockTimeoutVar))

	cmd.PersistentFlags().String(LockTableVar, DefaultLockTable, "DynamoDB table to write the lock in")
	viper.BindPFlag(LockTableVar, cmd.PersistentFlags().Lookup(LockTableVar))

	cmd.PersistentFlags().String(LockKeyNameVar, DefaultLockKeyName, "Name of the column where we write locks")
	viper.BindPFlag(LockKeyNameVar, cmd.PersistentFlags().Lookup(LockKeyNameVar))

	cmd.PersistentFlags().String(LockNameVar, DefaultLockName, "Name of the lock")
	viper.BindPFlag(LockNameVar, cmd.PersistentFlags().Lookup(LockNameVar))

	return cmd
}

func unlock() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unlock",
		Short: "Release a lock",
		Run: func(_ *cobra.Command, _ []string) {
			LockTable := viper.GetString(LockTableVar)
			LockKeyName := viper.GetString(LockKeyNameVar)
			LockName := viper.GetString(LockNameVar)

			svc := dynamodb.New(session.Must(session.NewSession()))

			log.Println("Acquiring lock to release it")
			output, err := svc.GetItem(&dynamodb.GetItemInput{
				TableName:      aws.String(LockTable),
				ConsistentRead: aws.Bool(true),
				Key: map[string]*dynamodb.AttributeValue{
					LockKeyName: {
						S: aws.String(LockName),
					},
				},
			})
			if err != nil {
				log.Fatalf("Failed to get lock during unlock process: %+v", err)
			}

			if len(output.Item) != 0 {
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
					log.Fatalf("Failed to delete lock: %+v", err)
				}
			}
		},
	}

	cmd.PersistentFlags().String(LockTableVar, DefaultLockTable, "DynamoDB table to delete the lock from")
	viper.BindPFlag(LockTableVar, cmd.PersistentFlags().Lookup(LockTableVar))

	cmd.PersistentFlags().String(LockKeyNameVar, DefaultLockKeyName, "Name of the column where we write locks")
	viper.BindPFlag(LockKeyNameVar, cmd.PersistentFlags().Lookup(LockKeyNameVar))

	cmd.PersistentFlags().String(LockNameVar, DefaultLockName, "Name of the lock")
	viper.BindPFlag(LockNameVar, cmd.PersistentFlags().Lookup(LockNameVar))

	return cmd
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "github-action-locks",
		Short: "Create a distributed lock for a GitHub Action",
	}

	viper.SetEnvPrefix("INPUT")
	viper.AutomaticEnv()

	rootCmd.AddCommand(lock())
	rootCmd.AddCommand(unlock())
	rootCmd.Execute()
}
