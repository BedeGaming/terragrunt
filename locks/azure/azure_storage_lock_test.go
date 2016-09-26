package azure

import (
	"fmt"
	"math/rand"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/stretchr/testify/assert"
)

func TestConfigMissingStorageAccountName(t *testing.T) {
	t.Parallel()

	conf := map[string]interface{}{}

	_, err := New(conf)
	assert.NotNil(t, err)
}

func TestConfigMissingContainerName(t *testing.T) {
	t.Parallel()

	conf := map[string]interface{}{
		"storage_account_name": "account",
	}

	_, err := New(conf)
	assert.NotNil(t, err)
}

func TestConfigMissingKey(t *testing.T) {
	t.Parallel()

	conf := map[string]interface{}{
		"storage_account_name": "account",
		"container_name":       "container",
	}

	_, err := New(conf)
	assert.NotNil(t, err)
}

func TestConfigValid(t *testing.T) {
	t.Parallel()

	conf := map[string]interface{}{
		"storage_account_name": "account",
		"container_name":       "container",
		"key":                  "key",
	}

	lock, err := New(conf)
	assert.NotNil(t, lock)
	assert.IsType(t, &StorageLock{}, lock)
	assert.Nil(t, err)

	storageLock := lock.(*StorageLock)
	assert.Equal(t, "account", storageLock.StorageAccountName)
	assert.Equal(t, "container", storageLock.ContainerName)
	assert.Equal(t, "key", storageLock.Key)
}

func TestAcquireLockContainerNotFoundError(t *testing.T) {
	t.Parallel()

	err := setupAzureAccTest(func(storageAccount, container, key string, client storage.BlobStorageClient) {
		lock := StorageLock{
			StorageAccountName: storageAccount,
			ContainerName:      "bad-container-name",
			Key:                key,
		}

		err := lock.AcquireLock()
		assert.NotNil(t, err)
		assert.Regexp(t, "does not exist", err.Error())
	})

	assert.Nil(t, err)
}

func TestAcquireAndReleaseLock(t *testing.T) {
	t.Parallel()

	err := setupAzureAccTest(func(storageAccount, container, key string, client storage.BlobStorageClient) {
		lock := StorageLock{
			StorageAccountName: storageAccount,
			ContainerName:      container,
			Key:                key,
		}

		// acquire, confirm file is leased
		err := lock.AcquireLock()
		assert.Nil(t, err)
		properties, err := client.GetBlobProperties(container, key)
		assert.Nil(t, err)
		assert.Equal(t, "locked", properties.LeaseStatus)

		// release, confirm lease has been released
		err = lock.ReleaseLock()
		assert.Nil(t, err)
		properties, err = client.GetBlobProperties(container, key)
		assert.Nil(t, err)
		assert.Equal(t, "unlocked", properties.LeaseStatus)
	})

	assert.Nil(t, err)
}

func TestAcquireLockAlreadyLockedError(t *testing.T) {
	t.Parallel()

	err := setupAzureAccTest(func(storageAccount, container, key string, client storage.BlobStorageClient) {
		lock := StorageLock{
			StorageAccountName: storageAccount,
			ContainerName:      container,
			Key:                key,
		}

		err := lock.AcquireLock()
		assert.Nil(t, err)

		err = lock.AcquireLock()
		assert.NotNil(t, err)

		// cleanup
		err = lock.ReleaseLock()
		assert.Nil(t, err)
	})

	assert.Nil(t, err)
}

func TestAcquireLockConcurrency(t *testing.T) {
	t.Parallel()

	err := setupAzureAccTest(func(storageAccount, container, key string, client storage.BlobStorageClient) {
		concurrency := 20
		lock := StorageLock{
			StorageAccountName: storageAccount,
			ContainerName:      container,
			Key:                key,
		}

		// Use a WaitGroup to ensure the test doesn't exit before all goroutines finish.
		var waitGroup sync.WaitGroup
		// This will count how many of the goroutines were able to acquire a lock. We use Go's atomic package to
		// ensure all modifications to this counter are atomic operations.
		locksAcquired := int32(0)

		// Launch a bunch of goroutines who will all try to acquire the lock at more or less the same time.
		// Only one should succeed.
		for i := 0; i < concurrency; i++ {
			waitGroup.Add(1)
			go func() {
				defer waitGroup.Done()
				err := lock.AcquireLock()
				if err == nil {
					atomic.AddInt32(&locksAcquired, 1)
				}
			}()
		}

		waitGroup.Wait()

		assert.Equal(t, int32(1), locksAcquired, "Only one of the goroutines should have been able to acquire a lock")
	})

	assert.Nil(t, err)
}

func setupAzureAccTest(testFunc func(storageAccount, container, key string, client storage.BlobStorageClient)) error {
	storageAccount := os.Getenv("ARM_STORAGE_ACCOUNT")
	if storageAccount == "" {
		return fmt.Errorf("ARM_STORAGE_ACCOUNT must be set for Azure lock tests")
	}

	container := randomBlobName(8)

	// uses ARM_ prefix to match Terraform
	accessKey := os.Getenv("ARM_ACCESS_KEY")
	if accessKey == "" {
		return fmt.Errorf("ARM_ACCESS_KEY must be set for Azure lock tests")
	}

	client, err := storage.NewBasicClient(storageAccount, accessKey)
	if err != nil {
		return err
	}
	blobClient := client.GetBlobService()

	_, err = client.GetBlobService().CreateContainerIfNotExists(container, storage.ContainerAccessTypePrivate)
	if err != nil {
		return err
	}

	name := randomBlobName(10)

	if err := blobClient.CreateBlockBlob(container, name); err != nil {
		return err
	}

	testFunc(storageAccount, container, name, blobClient)

	_, err = blobClient.DeleteContainerIfExists(container)
	return err
}

func randomBlobName(strlen int) string {
	rand.Seed(time.Now().UTC().UnixNano())
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, strlen)
	for i := 0; i < strlen; i++ {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}
