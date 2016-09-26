package config

import (
	"reflect"
	"testing"

	"github.com/gruntwork-io/terragrunt/errors"
	"github.com/gruntwork-io/terragrunt/locks/dynamodb"
	"github.com/gruntwork-io/terragrunt/remote"
	"github.com/stretchr/testify/assert"
)

func TestParseTerragruntConfigDynamoLockMinimalConfig(t *testing.T) {
	t.Parallel()

	config :=
		`
	lock = {
      backend = "dynamodb"
      config {
	    state_file_id = "expected-state-file-id"
      }
	}
	`

	terragruntConfig, err := parseConfigString(config)
	assert.Nil(t, err)

	assert.Nil(t, terragruntConfig.RemoteState)
	assert.NotNil(t, terragruntConfig.Lock)
	assert.IsType(t, &dynamodb.DynamoDbLock{}, terragruntConfig.Lock)
	lock := terragruntConfig.Lock.(*dynamodb.DynamoDbLock)
	assert.Equal(t, "expected-state-file-id", lock.StateFileId)
	assert.Equal(t, dynamodb.DEFAULT_AWS_REGION, lock.AwsRegion)
	assert.Equal(t, dynamodb.DEFAULT_TABLE_NAME, lock.TableName)
	assert.Equal(t, dynamodb.DEFAULT_MAX_RETRIES_WAITING_FOR_LOCK, lock.MaxLockRetries)
}

func TestParseTerragruntConfigDynamoLockFullConfig(t *testing.T) {
	t.Parallel()

	config :=
		`
	lock = {
      backend = "dynamodb"
      config {
	    state_file_id = "expected-state-file-id"
	    aws_region = "expected-region"
	    table_name = "expected-table-name"
	    max_lock_retries = 100
      }
	}
	`

	terragruntConfig, err := parseConfigString(config)
	assert.Nil(t, err)

	assert.Nil(t, terragruntConfig.RemoteState)
	assert.NotNil(t, terragruntConfig.Lock)
	assert.IsType(t, &dynamodb.DynamoDbLock{}, terragruntConfig.Lock)
	lock := terragruntConfig.Lock.(*dynamodb.DynamoDbLock)
	assert.Equal(t, "expected-state-file-id", lock.StateFileId)
	assert.Equal(t, "expected-region", lock.AwsRegion)
	assert.Equal(t, "expected-table-name", lock.TableName)
	assert.Equal(t, 100, lock.MaxLockRetries)
}

func TestParseTerragruntConfigDynamoLockMissingStateFileId(t *testing.T) {
	t.Parallel()

	config := `
    lock = {
        backend = "dynamodb"
        config {
        }
    }
	`

	_, err := parseConfigString(config)
	assert.EqualError(t, err, "unable to configure lock dynamodb: state_file_id cannot be empty")
}

func TestParseTerragruntConfigRemoteStateMinimalConfig(t *testing.T) {
	t.Parallel()

	config :=
		`
	remoteState = {
	  backend = "s3"
	}
	`

	terragruntConfig, err := parseConfigString(config)
	assert.Nil(t, err)

	assert.Nil(t, terragruntConfig.Lock)
	assert.NotNil(t, terragruntConfig.RemoteState)
	assert.Equal(t, "s3", terragruntConfig.RemoteState.Backend)
	assert.Empty(t, terragruntConfig.RemoteState.BackendConfigs)
}

func TestParseTerragruntConfigRemoteStateMissingBackend(t *testing.T) {
	t.Parallel()

	config :=
		`
	remoteState = {
	}
	`

	_, err := parseConfigString(config)
	assert.True(t, errors.IsError(err, remote.RemoteBackendMissing), "Unexpected error of type %s: %s", reflect.TypeOf(err), err)
}

func TestParseTerragruntConfigRemoteStateFullConfig(t *testing.T) {
	t.Parallel()

	config :=
		`
	remoteState = {
	  backend = "s3"
	  backendConfigs = {
	    encrypt = "true"
	    bucket = "my-bucket"
	    key = "terraform.tfstate"
	    region = "us-east-1"
	  }
	}
	`

	terragruntConfig, err := parseConfigString(config)
	assert.Nil(t, err)

	assert.Nil(t, terragruntConfig.Lock)
	assert.NotNil(t, terragruntConfig.RemoteState)
	assert.Equal(t, "s3", terragruntConfig.RemoteState.Backend)
	assert.NotEmpty(t, terragruntConfig.RemoteState.BackendConfigs)
	assert.Equal(t, "true", terragruntConfig.RemoteState.BackendConfigs["encrypt"])
	assert.Equal(t, "my-bucket", terragruntConfig.RemoteState.BackendConfigs["bucket"])
	assert.Equal(t, "terraform.tfstate", terragruntConfig.RemoteState.BackendConfigs["key"])
	assert.Equal(t, "us-east-1", terragruntConfig.RemoteState.BackendConfigs["region"])
}

func TestParseTerragruntConfigRemoteStateAndDynamoDbFullConfig(t *testing.T) {
	t.Parallel()

	config :=
		`
	lock = {
      backend = "dynamodb"
      config {
	    state_file_id = "expected-state-file-id"
	    aws_region = "expected-region"
	    table_name = "expected-table-name"
	    max_lock_retries = 100
      }
	}

	remoteState = {
	  backend = "s3"
	  backendConfigs = {
	    encrypt = "true"
	    bucket = "my-bucket"
	    key = "terraform.tfstate"
	    region = "us-east-1"
	  }
	}
	`

	terragruntConfig, err := parseConfigString(config)
	assert.Nil(t, err)

	assert.NotNil(t, terragruntConfig.Lock)
	assert.IsType(t, &dynamodb.DynamoDbLock{}, terragruntConfig.Lock)
	lock := terragruntConfig.Lock.(*dynamodb.DynamoDbLock)
	assert.Equal(t, "expected-state-file-id", lock.StateFileId)
	assert.Equal(t, "expected-region", lock.AwsRegion)
	assert.Equal(t, "expected-table-name", lock.TableName)
	assert.Equal(t, 100, lock.MaxLockRetries)

	assert.NotNil(t, terragruntConfig.RemoteState)
	assert.Equal(t, "s3", terragruntConfig.RemoteState.Backend)
	assert.NotEmpty(t, terragruntConfig.RemoteState.BackendConfigs)
	assert.Equal(t, "true", terragruntConfig.RemoteState.BackendConfigs["encrypt"])
	assert.Equal(t, "my-bucket", terragruntConfig.RemoteState.BackendConfigs["bucket"])
	assert.Equal(t, "terraform.tfstate", terragruntConfig.RemoteState.BackendConfigs["key"])
	assert.Equal(t, "us-east-1", terragruntConfig.RemoteState.BackendConfigs["region"])
}

func TestParseTerragruntConfigInvalidLockBackend(t *testing.T) {
	t.Parallel()

	config := `
    lock = {
        backend = "invalid"
        config {
        }
    }
	`

	_, err := parseConfigString(config)
	assert.EqualError(t, err, "unable to configure lock invalid: no Lock implementation found for invalid")
}

func TestParseTerragruntConfigEmptyConfig(t *testing.T) {
	t.Parallel()

	config := ``

	terragruntConfig, err := parseConfigString(config)
	assert.Nil(t, err)

	assert.Nil(t, terragruntConfig.RemoteState)
	assert.Nil(t, terragruntConfig.Lock)
}
