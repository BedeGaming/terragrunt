package azure

import (
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/gruntwork-io/terragrunt/errors"
	"github.com/gruntwork-io/terragrunt/locks"
	"github.com/gruntwork-io/terragrunt/util"
)

// StorageLock provides a lock backed by Azure Storage
// Validate must be called prior to the lock being used
type StorageLock struct {
	StorageAccountName string
	ContainerName      string
	Key                string
}

// New is the factory function for StorageLock
func New(conf map[string]string) (locks.Lock, error) {
	if conf["storage_account_name"] == "" {
		return nil, errors.WithStackTrace(fmt.Errorf("storage_account_name must be set"))
	}

	if conf["container_name"] == "" {
		return nil, errors.WithStackTrace(fmt.Errorf("container_name must be set"))
	}

	if conf["key"] == "" {
		return nil, errors.WithStackTrace(fmt.Errorf("key must be set"))
	}

	return &StorageLock{
		StorageAccountName: conf["storage_account_name"],
		ContainerName:      conf["container_name"],
		Key:                conf["key"],
	}, nil
}

// AcquireLock attempts to create a Blob in the Storage Container
func (lock *StorageLock) AcquireLock() error {
	util.Logger.Printf("Attempting to acquire lock for Blob key %s", lock.Key)

	client, err := lock.createStorageClient()
	if err != nil {
		return err
	}

	if err = client.CreateBlockBlob(lock.ContainerName, lock.Key); err != nil {
		return err
	}

	util.Logger.Printf("Lock acquired!")
	return nil
}

// ReleaseLock attempts to delete the Blob in the Storage Container
func (lock *StorageLock) ReleaseLock() error {
	util.Logger.Printf("Attempting to release lock for Blob key %s", lock.Key)

	client, err := lock.createStorageClient()
	if err != nil {
		return err
	}

	if _, err = client.DeleteBlobIfExists(lock.ContainerName, lock.Key, nil); err != nil {
		return err
	}

	util.Logger.Printf("Lock released!")
	return nil
}

// String returns a description of this lock
func (lock *StorageLock) String() string {
	return fmt.Sprintf("AzureStorageLock lock for state file %s", lock.Key)
}

// createStorageClient creates a new Blob Storage Client from the Azure SDK
// returns and error if ARM_ACCESS_KEY is empty
func (lock *StorageLock) createStorageClient() (*storage.BlobStorageClient, error) {
	accessKey := os.Getenv("ARM_ACCESS_KEY")
	if accessKey == "" {
		return nil, errors.WithStackTrace(fmt.Errorf("Error finding access key, did you forget to set ARM_ACCESS_KEY?"))
	}

	client, err := storage.NewBasicClient(lock.StorageAccountName, accessKey)
	if err != nil {
		return nil, err
	}

	blobClient := client.GetBlobService()
	return &blobClient, nil
}
